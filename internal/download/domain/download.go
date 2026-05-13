package domain

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"pb_launcher/helpers/unzip"
	"pb_launcher/internal/download/domain/dtos"
	"pb_launcher/internal/download/domain/repositories"
	"pb_launcher/internal/download/domain/services"
	"sync"
)

type DownloadUsecase struct {
	service         services.ReleaseVersionsService
	repository      repositories.ReleaseRepository
	artifactStorage services.RepositoryArtifactStorage
	unzip           *unzip.Unzip
	mu              sync.Mutex
}

func NewDownloadUsecase(
	service services.ReleaseVersionsService,
	repository repositories.ReleaseRepository,
	artifactStorage services.RepositoryArtifactStorage,
	unzip *unzip.Unzip,
) *DownloadUsecase {
	return &DownloadUsecase{
		service:         service,
		repository:      repository,
		artifactStorage: artifactStorage,
		unzip:           unzip,
	}
}

func (uc *DownloadUsecase) syncRepository(ctx context.Context, repo dtos.Repository) error {
	availableReleases, err := uc.service.FetchReleases(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to fetch releases: %w", err)
	}

	existingReleases, err := uc.repository.ListReleases(ctx, repo.ID)
	if err != nil {
		return fmt.Errorf("failed to list existing releases: %w", err)
	}

	diff := uc.DiffReleases(availableReleases, existingReleases)
	if len(diff) > 0 {
		if err := uc.repository.SaveReleases(ctx, diff); err != nil {
			return fmt.Errorf("failed to save new releases: %w", err)
		}
	}

	if err := uc.resolveMissingReleases(ctx, repo); err != nil {
		return fmt.Errorf("failed to resolve missing releases: %w", err)
	}
	return nil
}

func (uc *DownloadUsecase) SyncRepository(ctx context.Context, repositoryID string) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	repo, err := uc.repository.FindRepository(ctx, repositoryID)
	if err != nil {
		return err
	}
	if err := uc.repository.MarkRepositorySyncing(ctx, repo.ID); err != nil {
		return err
	}

	if err := uc.syncRepository(ctx, *repo); err != nil {
		_ = uc.repository.MarkRepositorySyncError(ctx, repo.ID, err.Error())
		return err
	}
	return uc.repository.MarkRepositorySyncSuccess(ctx, repo.ID)
}

// diffReleases returns the releases present in 'a' but not in 'b'.
// Example:
// a = [{Version: 1.0.0}, {Version: 1.2.0}, {Version: 2.0.0}]
// b = [{Version: 1.0.0}, {Version: 2.0.0}]
// Result: [{Version: 1.2.0}]
func (uc *DownloadUsecase) DiffReleases(a []dtos.Release, b []dtos.Release) []dtos.Release {
	bVersions := make(map[string]struct{})
	for _, release := range b {
		if release.Version != nil {
			bVersions[release.Version.String()] = struct{}{}
		}
	}

	var diff []dtos.Release
	for _, release := range a {
		if release.Version != nil {
			if _, exists := bVersions[release.Version.String()]; !exists {
				diff = append(diff, release)
			}
		}
	}

	return diff
}

func (uc *DownloadUsecase) processDownload(ctx context.Context, repo dtos.Repository, release dtos.Release) error {
	zipPath, err := uc.service.Download(ctx, repo, release.ReleaseAsset)
	if err != nil {
		slog.Error("failed to download release", "error", err)
		return err
	}
	defer os.Remove(zipPath)

	tempDir, err := os.MkdirTemp("", "release-extract-*")
	if err != nil {
		slog.Error("failed to create temp directory for extraction", "error", err)
		return err
	}
	defer os.RemoveAll(tempDir)

	extractedPaths, err := uc.unzip.Extract(zipPath, tempDir)
	if err != nil {
		slog.Error("failed to extract release", "error", err, "zipPath", zipPath)
		return err
	}

	for _, relativePath := range extractedPaths {
		fullPath := filepath.Join(tempDir, relativePath)
		file, err := os.Open(fullPath)
		if err != nil {
			slog.Error("failed to open extracted file", "error", err, "path", fullPath)
			return err
		}
		defer file.Close()
		outFilePath := filepath.Join(release.Version.String(), relativePath)
		if _, err := uc.artifactStorage.Save(ctx, release.RepositoryID, outFilePath, file); err != nil {
			file.Close()
			slog.Error("failed to save extracted file", "error", err, "path", relativePath)
			return err
		}

		file.Close()
	}
	slog.Info("release downloaded successfully", "version", release.Version.String())
	return nil
}

func (uc *DownloadUsecase) EnsureReleaseDownloaded(ctx context.Context, releaseID string) error {
	release, err := uc.repository.FindRelease(ctx, releaseID)
	if err != nil {
		return err
	}
	repo, err := uc.repository.FindRepository(ctx, release.RepositoryID)
	if err != nil {
		return err
	}

	downloadedVersions, err := uc.artifactStorage.Versions(ctx, repo.ID)
	if err != nil {
		return err
	}
	for _, downloaded := range downloadedVersions {
		if downloaded.Equal(release.Version) {
			return nil
		}
	}

	return uc.processDownload(ctx, *repo, *release)
}

func (uc *DownloadUsecase) resolveMissingReleases(ctx context.Context, repo dtos.Repository) error {
	releases, err := uc.repository.ListReleases(ctx, repo.ID)
	if err != nil {
		slog.Error("failed to retrieve available releases", "error", err)
		return err
	}

	downloadedVersions, err := uc.artifactStorage.Versions(ctx, repo.ID)
	if err != nil {
		slog.Error("failed to retrieve downloaded versions", "error", err)
		return err
	}

	downloadedSet := make(map[string]struct{}, len(downloadedVersions))
	for _, version := range downloadedVersions {
		downloadedSet[version.String()] = struct{}{}
	}

	var pendingDownloads []dtos.Release
	for _, release := range releases {
		if _, exists := downloadedSet[release.Version.String()]; !exists {
			pendingDownloads = append(pendingDownloads, release)
		}
	}

	for _, pending := range pendingDownloads {
		if err := uc.processDownload(ctx, repo, pending); err != nil {
			slog.Error("failed to process download", "version", pending.Version.String(), "error", err)
			return err
		}
	}

	return nil
}

func (uc *DownloadUsecase) Run(ctx context.Context) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	repositories, err := uc.repository.ListRepositories(ctx)
	if err != nil {
		slog.Error("failed to list repositories", "error", err)
		return err
	}
	var combinedErr error
	for _, repo := range repositories {
		if err := uc.repository.MarkRepositorySyncing(ctx, repo.ID); err != nil {
			combinedErr = errors.Join(combinedErr, err)
			continue
		}
		if err := uc.syncRepository(ctx, repo); err != nil {
			slog.Error("failed to sync repository", "repository_id", repo.ID, "error", err)
			_ = uc.repository.MarkRepositorySyncError(ctx, repo.ID, err.Error())
			combinedErr = errors.Join(combinedErr, err)
			continue
		}
		if err := uc.repository.MarkRepositorySyncSuccess(ctx, repo.ID); err != nil {
			combinedErr = errors.Join(combinedErr, err)
		}
	}
	return combinedErr
}

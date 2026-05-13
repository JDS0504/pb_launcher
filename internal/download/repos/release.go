package repos

import (
	"context"
	"fmt"
	"log/slog"
	"pb_launcher/collections"
	"pb_launcher/internal/download/domain/dtos"
	"pb_launcher/internal/download/domain/repositories"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type ReleaseRepository struct {
	app *pocketbase.PocketBase
}

var _ repositories.ReleaseRepository = (*ReleaseRepository)(nil)

func NewReleaseRepository(app *pocketbase.PocketBase) *ReleaseRepository {
	return &ReleaseRepository{app: app}
}

func (r *ReleaseRepository) ListRepositories(ctx context.Context) ([]dtos.Repository, error) {
	records, err := r.app.FindAllRecords(collections.Repositories, dbx.NewExp("disabled = false"))
	if err != nil {
		slog.Error("Failed to fetch repositories from database", "error", err)
		return nil, err
	}

	repositories := make([]dtos.Repository, 0, len(records))
	for _, record := range records {
		releasePattern := strings.TrimSpace(record.GetString("release_file_pattern"))
		if releasePattern == "" {
			slog.Warn("Release file pattern is empty, skipping record", "record_id", record.Id)
			continue
		}

		releasePatternRegex, err := regexp.Compile(releasePattern)
		if err != nil {
			slog.Warn("Invalid release file pattern regex", "pattern", releasePattern, "record_id", record.Id, "error", err)
			continue
		}

		execPattern := strings.TrimSpace(record.GetString("exec_file_pattern"))
		if execPattern == "" {
			slog.Warn("Exec file pattern is empty, skipping record", "record_id", record.Id)
			continue
		}

		execPatternRegex, err := regexp.Compile(execPattern)
		if err != nil {
			slog.Warn("Invalid exec file pattern regex", "pattern", execPattern, "record_id", record.Id, "error", err)
			continue
		}

		retention := max(record.GetInt("retention"), 1)
		retention = min(retention, 6)

		repositories = append(repositories, dtos.Repository{
			ID:                 record.Id,
			Repo:               record.GetString("repository"),
			Token:              record.GetString("token"),
			ReleaseFilePattern: releasePatternRegex,
			ExecFilePattern:    execPatternRegex,
			Retention:          retention,
		})
	}

	return repositories, nil
}

func (r *ReleaseRepository) FindRepository(ctx context.Context, repositoryID string) (*dtos.Repository, error) {
	record, err := r.app.FindRecordById(collections.Repositories, repositoryID)
	if err != nil {
		return nil, err
	}
	if record.GetBool("disabled") {
		return nil, fmt.Errorf("repository %s is disabled", repositoryID)
	}

	releasePattern := strings.TrimSpace(record.GetString("release_file_pattern"))
	releasePatternRegex, err := regexp.Compile(releasePattern)
	if err != nil {
		return nil, fmt.Errorf("invalid release file pattern: %w", err)
	}
	execPattern := strings.TrimSpace(record.GetString("exec_file_pattern"))
	execPatternRegex, err := regexp.Compile(execPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid exec file pattern: %w", err)
	}

	retention := max(record.GetInt("retention"), 1)
	retention = min(retention, 6)
	return &dtos.Repository{
		ID:                 record.Id,
		Repo:               record.GetString("repository"),
		Token:              record.GetString("token"),
		ReleaseFilePattern: releasePatternRegex,
		ExecFilePattern:    execPatternRegex,
		Retention:          retention,
	}, nil
}

func (r *ReleaseRepository) ListReleases(ctx context.Context, repositoryId string) ([]dtos.Release, error) {
	records, err := r.app.FindAllRecords(collections.Releases,
		dbx.NewExp("repository={:id}", dbx.Params{"id": repositoryId}),
	)
	if err != nil {
		slog.Error("failed to fetch releases from database", "error", err)
		return nil, err
	}

	releases := make([]dtos.Release, 0, len(records))

	for _, record := range records {
		versionString := record.GetString("version")
		v, err := version.NewVersion(versionString)
		if err != nil {
			slog.Warn("invalid version format", "version", versionString, "error", err)
			continue
		}

		releases = append(releases, dtos.Release{
			RepositoryID: record.GetString("repository"),
			Version:      v,
			ReleaseName:  record.GetString("release_name"),
			PublishedAt:  record.GetDateTime("published_at").Time(),
			ReleaseAsset: dtos.ReleaseAsset{
				AssetID:       record.GetString("asset_id"),
				AssetFileName: record.GetString("asset_file_name"),
				DownloadURL:   record.GetString("download_url"),
				AssetSize:     int64(record.GetInt("asset_size")),
			},
		})
	}
	return releases, nil
}

func (r *ReleaseRepository) FindRelease(ctx context.Context, releaseID string) (*dtos.Release, error) {
	record, err := r.app.FindRecordById(collections.Releases, releaseID)
	if err != nil {
		return nil, err
	}
	versionString := record.GetString("version")
	v, err := version.NewVersion(versionString)
	if err != nil {
		return nil, fmt.Errorf("invalid version format %q: %w", versionString, err)
	}

	return &dtos.Release{
		RepositoryID: record.GetString("repository"),
		Version:      v,
		ReleaseName:  record.GetString("release_name"),
		PublishedAt:  record.GetDateTime("published_at").Time(),
		ReleaseAsset: dtos.ReleaseAsset{
			AssetID:       record.GetString("asset_id"),
			AssetFileName: record.GetString("asset_file_name"),
			DownloadURL:   record.GetString("download_url"),
			AssetSize:     int64(record.GetInt("asset_size")),
		},
	}, nil
}

func (r *ReleaseRepository) SaveReleases(ctx context.Context, releases []dtos.Release) error {
	if len(releases) == 0 {
		slog.Info("no new releases to insert")
		return nil
	}

	collection, err := r.app.FindCollectionByNameOrId(collections.Releases)
	if err != nil {
		slog.Error("failed to find releases collection", "error", err)
		return err
	}

	for _, release := range releases {
		record := core.NewRecord(collection)
		record.Set("repository", release.RepositoryID)
		record.Set("version", release.Version.String())
		record.Set("release_name", release.ReleaseName)
		record.Set("published_at", release.PublishedAt)
		record.Set("asset_file_name", release.AssetFileName)
		record.Set("asset_id", release.AssetID)
		record.Set("download_url", release.DownloadURL)
		record.Set("asset_size", release.AssetSize)

		if err := r.app.Save(record); err != nil {
			slog.Error("failed to save release record", "version", release.Version.String(), "error", err)
			return err
		}
	}
	return nil
}

func (r *ReleaseRepository) MarkRepositorySyncing(ctx context.Context, repositoryID string) error {
	record, err := r.app.FindRecordById(collections.Repositories, repositoryID)
	if err != nil {
		return err
	}
	record.Set("last_sync_status", "syncing")
	record.Set("last_sync_error", nil)
	return r.app.Save(record)
}

func (r *ReleaseRepository) MarkRepositorySyncSuccess(ctx context.Context, repositoryID string) error {
	record, err := r.app.FindRecordById(collections.Repositories, repositoryID)
	if err != nil {
		return err
	}
	record.Set("last_sync_status", "success")
	record.Set("last_sync_error", nil)
	record.Set("last_sync_at", time.Now())
	return r.app.Save(record)
}

func (r *ReleaseRepository) MarkRepositorySyncError(ctx context.Context, repositoryID string, errorMessage string) error {
	record, err := r.app.FindRecordById(collections.Repositories, repositoryID)
	if err != nil {
		return err
	}
	record.Set("last_sync_status", "error")
	record.Set("last_sync_error", errorMessage)
	record.Set("last_sync_at", time.Now())
	return r.app.Save(record)
}

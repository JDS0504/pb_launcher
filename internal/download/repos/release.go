package repos

import (
	"context"
	"fmt"
	"log/slog"
	"pb_launcher/collections"
	"pb_launcher/internal/download/domain/dtos"
	"pb_launcher/internal/download/domain/repositories"
	"regexp"

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
	releasePatternRegex, err := regexp.Compile(`pocketbase_.+_linux_amd64\.zip`)
	if err != nil {
		return nil, err
	}
	execPatternRegex, err := regexp.Compile(`^pocketbase`)
	if err != nil {
		return nil, err
	}

	return []dtos.Repository{
		{
			ID:                 "pb91u2l315h29a5",
			Repo:               "pocketbase/pocketbase",
			Token:              "",
			ReleaseFilePattern: releasePatternRegex,
			ExecFilePattern:    execPatternRegex,
			Retention:          3,
		},
	}, nil
}

func (r *ReleaseRepository) FindRepository(ctx context.Context, repositoryID string) (*dtos.Repository, error) {
	if repositoryID != "pb91u2l315h29a5" {
		return nil, fmt.Errorf("repository %s not found", repositoryID)
	}

	releasePatternRegex, err := regexp.Compile(`pocketbase_.+_linux_amd64\.zip`)
	if err != nil {
		return nil, err
	}
	execPatternRegex, err := regexp.Compile(`^pocketbase`)
	if err != nil {
		return nil, err
	}

	return &dtos.Repository{
		ID:                 "pb91u2l315h29a5",
		Repo:               "pocketbase/pocketbase",
		Token:              "",
		ReleaseFilePattern: releasePatternRegex,
		ExecFilePattern:    execPatternRegex,
		Retention:          3,
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
	return nil
}

func (r *ReleaseRepository) MarkRepositorySyncSuccess(ctx context.Context, repositoryID string) error {
	return nil
}

func (r *ReleaseRepository) MarkRepositorySyncError(ctx context.Context, repositoryID string, errorMessage string) error {
	return nil
}

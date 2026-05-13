package repositories

import (
	"context"
	"pb_launcher/internal/download/domain/dtos"
)

type ReleaseRepository interface {
	ListRepositories(ctx context.Context) ([]dtos.Repository, error)
	FindRepository(ctx context.Context, repositoryID string) (*dtos.Repository, error)
	ListReleases(ctx context.Context, repositoryID string) ([]dtos.Release, error)
	FindRelease(ctx context.Context, releaseID string) (*dtos.Release, error)
	SaveReleases(ctx context.Context, releases []dtos.Release) error
	MarkRepositorySyncing(ctx context.Context, repositoryID string) error
	MarkRepositorySyncSuccess(ctx context.Context, repositoryID string) error
	MarkRepositorySyncError(ctx context.Context, repositoryID string, errorMessage string) error
}

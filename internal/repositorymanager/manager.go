package repositorymanager

import (
	"context"
	"pb_launcher/collections"
	download "pb_launcher/internal/download/domain"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

type RepositoryStatus struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Repository              string `json:"repository"`
	Token                   string `json:"token"`
	Retention               int    `json:"retention"`
	ReleaseFilePattern      string `json:"release_file_pattern"`
	ExecFilePattern         string `json:"exec_file_pattern"`
	Disabled                bool   `json:"disabled"`
	LastSyncAt              string `json:"last_sync_at"`
	LastSyncStatus          string `json:"last_sync_status"`
	LastSyncError           string `json:"last_sync_error"`
	ReleaseCount            int    `json:"release_count"`
	DownloadedVersionsCount int    `json:"downloaded_versions_count"`
}

type Manager struct {
	app        *pocketbase.PocketBase
	downloader *download.DownloadUsecase
}

func NewManager(app *pocketbase.PocketBase, downloader *download.DownloadUsecase) *Manager {
	return &Manager{app: app, downloader: downloader}
}

func (m *Manager) ListStatus(ctx context.Context) ([]RepositoryStatus, error) {
	records, err := m.app.FindAllRecords(collections.Repositories)
	if err != nil {
		return nil, err
	}
	statuses := make([]RepositoryStatus, 0, len(records))
	for _, record := range records {
		releaseCount, err := m.app.CountRecords(collections.Releases, dbx.NewExp("repository={:id}", dbx.Params{"id": record.Id}))
		if err != nil {
			return nil, err
		}
		status := record.GetString("last_sync_status")
		if status == "" {
			status = "never"
		}
		statuses = append(statuses, RepositoryStatus{
			ID:                      record.Id,
			Name:                    record.GetString("name"),
			Repository:              record.GetString("repository"),
			Token:                   record.GetString("token"),
			Retention:               record.GetInt("retention"),
			ReleaseFilePattern:      record.GetString("release_file_pattern"),
			ExecFilePattern:         record.GetString("exec_file_pattern"),
			Disabled:                record.GetBool("disabled"),
			LastSyncAt:              record.GetDateTime("last_sync_at").String(),
			LastSyncStatus:          status,
			LastSyncError:           record.GetString("last_sync_error"),
			ReleaseCount:            int(releaseCount),
			DownloadedVersionsCount: 0,
		})
	}
	return statuses, nil
}

func (m *Manager) Sync(ctx context.Context, repositoryID string) error {
	return m.downloader.SyncRepository(ctx, repositoryID)
}

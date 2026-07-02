package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"pb_launcher/collections"
	"pb_launcher/configs"
	"pb_launcher/helpers/unzip"
	ziphelper "pb_launcher/helpers/zip"
	download "pb_launcher/internal/download/domain"
	"pb_launcher/internal/launcher/domain/models"
	"pb_launcher/internal/launcher/domain/repositories"
	"pb_launcher/internal/operationlog"
	"pb_launcher/utils/domainutil"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const manifestFormat = "pblauncher-backup/v1"

type Manifest struct {
	Format    string          `json:"format"`
	CreatedAt time.Time       `json:"created_at"`
	Service   ManifestService `json:"service"`
}

type ManifestService struct {
	Name             string `json:"name"`
	ReleaseID        string `json:"release_id"`
	RepositoryID     string `json:"repository_id"`
	Version          string `json:"version"`
	RestartPolicy    string `json:"restart_policy"`
	BootUserEmail    string `json:"boot_user_email"`
	BootUserPassword string `json:"boot_user_password"`
}

type BackupFile struct {
	Path     string
	Filename string
}

type SnapshotInfo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	ServiceID     string    `json:"service_id"`
	SourceService string    `json:"source_service"`
	Version       string    `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	Size          int64     `json:"size"`
	Path          string    `json:"-"`
	MetadataPath  string    `json:"-"`
}

type Manager struct {
	app          *pocketbase.PocketBase
	dataDir      string
	domainBase   string
	serviceRepo  repositories.ServiceRepository
	commandsRepo repositories.CommandsRepository
	downloader   *download.DownloadUsecase
	zipper       *ziphelper.Zip
	unzipper     *unzip.Unzip
	logger       *operationlog.Logger
}

func NewManager(
	app *pocketbase.PocketBase,
	cfg configs.Config,
	serviceRepo repositories.ServiceRepository,
	commandsRepo repositories.CommandsRepository,
	downloader *download.DownloadUsecase,
	zipper *ziphelper.Zip,
	unzipper *unzip.Unzip,
	logger *operationlog.Logger,
) *Manager {
	return &Manager{
		app:          app,
		dataDir:      cfg.GetDataDir(),
		domainBase:   cfg.GetDomain(),
		serviceRepo:  serviceRepo,
		commandsRepo: commandsRepo,
		downloader:   downloader,
		zipper:       zipper,
		unzipper:     unzipper,
		logger:       logger,
	}
}

func (m *Manager) Create(ctx context.Context, serviceID string) (*BackupFile, error) {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	if service.Status != models.Stopped {
		return nil, fmt.Errorf("service must be stopped before backup")
	}

	serviceDir := filepath.Join(m.dataDir, service.Name)
	if info, err := os.Stat(serviceDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("service data directory not found")
	}

	manifest := Manifest{
		Format:    manifestFormat,
		CreatedAt: time.Now().UTC(),
		Service: ManifestService{
			Name:             service.Name,
			ReleaseID:        service.ReleaseID,
			RepositoryID:     service.RepositoryID,
			Version:          service.Version,
			RestartPolicy:    string(service.RestartPolicy),
			BootUserEmail:    service.BootUserEmail,
			BootUserPassword: service.BootUserPassword,
		},
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}

	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("pblauncher-%s-%d.zip", service.ID, time.Now().Unix()))
	if err := m.zipper.CreateFromDir(serviceDir, backupPath, "data", map[string][]byte{"manifest.json": manifestBytes}); err != nil {
		m.logger.Error(ctx, service.ID, "backup", err.Error(), nil)
		return nil, err
	}
	m.logger.Success(ctx, service.ID, "backup", "backup created successfully", nil)

	return &BackupFile{
		Path:     backupPath,
		Filename: fmt.Sprintf("%s-%s-backup.zip", sanitizeFilename(service.Name), service.ID),
	}, nil
}

func (m *Manager) snapshotsDir(serviceID string) string {
	return filepath.Join(m.dataDir, "_snapshots", serviceID)
}

func (m *Manager) ListSnapshots(ctx context.Context, serviceID string) ([]SnapshotInfo, error) {
	if _, err := m.serviceRepo.FindService(ctx, serviceID); err != nil {
		return nil, err
	}
	dir := m.snapshotsDir(serviceID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SnapshotInfo{}, nil
		}
		return nil, err
	}

	snapshots := []SnapshotInfo{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		metadataPath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(metadataPath)
		if err != nil {
			return nil, err
		}
		var snapshot SnapshotInfo
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return nil, err
		}
		snapshot.MetadataPath = metadataPath
		snapshot.Path = filepath.Join(dir, snapshot.ID+".zip")
		if info, err := os.Stat(snapshot.Path); err == nil {
			snapshot.Size = info.Size()
		}
		snapshots = append(snapshots, snapshot)
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})
	return snapshots, nil
}

func (m *Manager) CreateSnapshot(ctx context.Context, serviceID string, name string) (*SnapshotInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("snapshot name is required")
	}
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	if service.Status != models.Stopped {
		return nil, fmt.Errorf("service must be stopped before snapshot")
	}

	serviceDir := filepath.Join(m.dataDir, service.Name)
	if info, err := os.Stat(serviceDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("service data directory not found")
	}

	snapshotID := fmt.Sprintf("%d", time.Now().UnixNano())
	snapshotDir := m.snapshotsDir(service.ID)
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, err
	}

	manifest := Manifest{
		Format:    manifestFormat,
		CreatedAt: time.Now().UTC(),
		Service: ManifestService{
			Name:             service.Name,
			ReleaseID:        service.ReleaseID,
			RepositoryID:     service.RepositoryID,
			Version:          service.Version,
			RestartPolicy:    string(service.RestartPolicy),
			BootUserEmail:    service.BootUserEmail,
			BootUserPassword: service.BootUserPassword,
		},
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}

	snapshotPath := filepath.Join(snapshotDir, snapshotID+".zip")
	if err := m.zipper.CreateFromDir(serviceDir, snapshotPath, "data", map[string][]byte{"manifest.json": manifestBytes}); err != nil {
		m.logger.Error(ctx, service.ID, "snapshot_create", err.Error(), nil)
		return nil, err
	}
	info, _ := os.Stat(snapshotPath)
	snapshot := SnapshotInfo{
		ID:            snapshotID,
		Name:          name,
		ServiceID:     service.ID,
		SourceService: service.Name,
		Version:       service.Version,
		CreatedAt:     time.Now().UTC(),
		Path:          snapshotPath,
		MetadataPath:  filepath.Join(snapshotDir, snapshotID+".json"),
	}
	if info != nil {
		snapshot.Size = info.Size()
	}
	metadataBytes, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(snapshot.MetadataPath, metadataBytes, 0644); err != nil {
		return nil, err
	}
	m.logger.Success(ctx, service.ID, "snapshot_create", "snapshot created successfully", map[string]any{"snapshot_id": snapshot.ID, "name": snapshot.Name})
	return &snapshot, nil
}

func (m *Manager) RestoreSnapshot(ctx context.Context, serviceID string, snapshotID string, name string) (string, error) {
	snapshot, err := m.findSnapshot(ctx, serviceID, snapshotID)
	if err != nil {
		return "", err
	}
	restoredID, err := m.Restore(ctx, snapshot.Path, name)
	if err != nil {
		return "", err
	}
	m.logger.Success(ctx, restoredID, "snapshot_restore", "snapshot restored successfully", map[string]any{"snapshot_id": snapshot.ID, "source_service_id": serviceID})
	return restoredID, nil
}

func (m *Manager) DeleteSnapshot(ctx context.Context, serviceID string, snapshotID string) error {
	snapshot, err := m.findSnapshot(ctx, serviceID, snapshotID)
	if err != nil {
		return err
	}
	if err := os.Remove(snapshot.Path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(snapshot.MetadataPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	m.logger.Success(ctx, serviceID, "snapshot_delete", "snapshot deleted successfully", map[string]any{"snapshot_id": snapshot.ID})
	return nil
}

func (m *Manager) GetSnapshotFile(ctx context.Context, serviceID string, snapshotID string) (*BackupFile, error) {
	snapshot, err := m.findSnapshot(ctx, serviceID, snapshotID)
	if err != nil {
		return nil, err
	}
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	return &BackupFile{
		Path:     snapshot.Path,
		Filename: fmt.Sprintf("%s-%s-snapshot.zip", sanitizeFilename(service.Name), snapshot.ID),
	}, nil
}

func (m *Manager) findSnapshot(ctx context.Context, serviceID string, snapshotID string) (*SnapshotInfo, error) {
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" || strings.Contains(snapshotID, "/") || strings.Contains(snapshotID, "..") {
		return nil, fmt.Errorf("invalid snapshot id")
	}
	snapshots, err := m.ListSnapshots(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	for _, snapshot := range snapshots {
		if snapshot.ID == snapshotID {
			return &snapshot, nil
		}
	}
	return nil, fmt.Errorf("snapshot not found")
}

func (m *Manager) Restore(ctx context.Context, backupPath string, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("instance name is required")
	}

	tempDir, err := os.MkdirTemp("", "pblauncher-restore-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	if _, err := m.unzipper.Extract(backupPath, tempDir); err != nil {
		return "", err
	}

	manifest, err := readManifest(filepath.Join(tempDir, "manifest.json"))
	if err != nil {
		return "", err
	}
	if manifest.Format != manifestFormat {
		return "", fmt.Errorf("unsupported backup format %q", manifest.Format)
	}

	release, err := m.serviceRepo.FindRelease(ctx, manifest.Service.ReleaseID)
	if err != nil {
		return "", fmt.Errorf("backup release not found: %w", err)
	}
	if release.RepositoryID != manifest.Service.RepositoryID || release.Version != manifest.Service.Version {
		return "", fmt.Errorf("backup release metadata does not match local release record")
	}
	if err := m.downloader.EnsureReleaseDownloaded(ctx, manifest.Service.ReleaseID); err != nil {
		return "", fmt.Errorf("failed to download backup release: %w", err)
	}

	collection, err := m.app.FindCachedCollectionByNameOrId(collections.Services)
	if err != nil {
		return "", err
	}
	record := core.NewRecord(collection)
	record.Set("name", name)
	record.Set("release", manifest.Service.ReleaseID)
	record.Set("restart_policy", manifest.Service.RestartPolicy)
	record.Set("status", string(models.Restoring))
	record.Set("boot_user_email", manifest.Service.BootUserEmail)
	record.Set("boot_user_password", manifest.Service.BootUserPassword)

	if err := m.app.Save(record); err != nil {
		return "", err
	}

	serviceDir := filepath.Join(m.dataDir, record.GetString(name))
	if err := copyDir(filepath.Join(tempDir, "data"), serviceDir); err != nil {
		_ = m.app.Delete(record)
		_ = os.RemoveAll(serviceDir)
		return "", err
	}

	record.Set("status", string(models.Stopped))
	if err := m.app.Save(record); err != nil {
		_ = os.RemoveAll(serviceDir)
		return "", err
	}

	if err := m.createFriendlyDomain(record); err != nil {
		_ = m.app.Delete(record)
		_ = os.RemoveAll(serviceDir)
		return "", err
	}

	if err := m.commandsRepo.PublishStartCommand(ctx, record.Id); err != nil {
		m.logger.Error(ctx, record.Id, "restore", err.Error(), map[string]any{"source_version": manifest.Service.Version})
		return "", err
	}
	m.logger.Success(ctx, record.Id, "restore", "backup restored successfully", map[string]any{"source_version": manifest.Service.Version})

	return record.Id, nil
}

func (m *Manager) Clone(ctx context.Context, sourceServiceID string, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("instance name is required")
	}

	source, err := m.serviceRepo.FindService(ctx, sourceServiceID)
	if err != nil {
		return "", err
	}
	if source.Status != models.Stopped {
		return "", fmt.Errorf("service must be stopped before clone")
	}
	if err := m.downloader.EnsureReleaseDownloaded(ctx, source.ReleaseID); err != nil {
		return "", fmt.Errorf("failed to download service release: %w", err)
	}

	sourceDir := filepath.Join(m.dataDir, source.Name)
	if info, err := os.Stat(sourceDir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("service data directory not found")
	}

	collection, err := m.app.FindCachedCollectionByNameOrId(collections.Services)
	if err != nil {
		return "", err
	}
	record := core.NewRecord(collection)
	record.Set("name", name)
	record.Set("release", source.ReleaseID)
	record.Set("restart_policy", string(source.RestartPolicy))
	record.Set("status", string(models.Restoring))
	record.Set("boot_user_email", source.BootUserEmail)
	record.Set("boot_user_password", source.BootUserPassword)

	if err := m.app.Save(record); err != nil {
		return "", err
	}

	targetDir := filepath.Join(m.dataDir, record.GetString(name))
	if err := copyDir(sourceDir, targetDir); err != nil {
		_ = m.app.Delete(record)
		_ = os.RemoveAll(targetDir)
		return "", err
	}

	record.Set("status", string(models.Stopped))
	if err := m.app.Save(record); err != nil {
		_ = os.RemoveAll(targetDir)
		return "", err
	}

	if err := m.createFriendlyDomain(record); err != nil {
		_ = m.app.Delete(record)
		_ = os.RemoveAll(targetDir)
		return "", err
	}

	if err := m.commandsRepo.PublishStartCommand(ctx, record.Id); err != nil {
		m.logger.Error(ctx, record.Id, "clone", err.Error(), map[string]any{"source_service_id": source.ID})
		return "", err
	}
	m.logger.Success(ctx, record.Id, "clone", "service cloned successfully", map[string]any{"source_service_id": source.ID})

	return record.Id, nil
}

func readManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("manifest.json not found: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest.json: %w", err)
	}
	return &manifest, nil
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "service"
	}
	var b strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	return b.String()
}

func copyDir(source, destination string) error {
	cleanSource, err := filepath.Abs(source)
	if err != nil {
		return err
	}
	if info, err := os.Stat(cleanSource); err != nil || !info.IsDir() {
		return fmt.Errorf("backup data directory not found")
	}
	if err := os.MkdirAll(destination, 0755); err != nil {
		return err
	}

	return filepath.WalkDir(cleanSource, func(sourcePath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(cleanSource, sourcePath)
		if err != nil || strings.HasPrefix(relPath, "..") {
			return fmt.Errorf("illegal restore path detected: %s", sourcePath)
		}
		targetPath := filepath.Join(destination, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(sourcePath, targetPath, info.Mode())
	})
}

func copyFile(source, destination string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func (m *Manager) createFriendlyDomain(serviceRecord *core.Record) error {
	friendlyDomain, err := domainutil.GenerateFriendlyDomain(serviceRecord.GetString("name"), m.domainBase)
	if err != nil {
		return nil
	}

	domainCollection, err := m.app.FindCachedCollectionByNameOrId(collections.ServicesDomains)
	if err != nil {
		return err
	}

	existing, err := m.app.FindFirstRecordByFilter(
		collections.ServicesDomains,
		"domain = {:domain}",
		map[string]any{"domain": friendlyDomain},
	)
	if err == nil && existing != nil {
		serviceId := existing.GetString("service")
		isOrphanOrDeleted := false
		if serviceId != "" {
			serviceRecord, err := m.app.FindRecordById(collections.Services, serviceId)
			if err != nil || serviceRecord == nil {
				isOrphanOrDeleted = true
			} else {
				serviceDeleted := serviceRecord.GetDateTime("deleted")
				if !serviceDeleted.IsZero() {
					isOrphanOrDeleted = true
				}
			}
		} else {
			isOrphanOrDeleted = true
		}

		if isOrphanOrDeleted {
			_ = m.app.Delete(existing)
		} else {
			return fmt.Errorf("el nombre '%s' no está disponible porque el dominio '%s' ya está en uso", serviceRecord.GetString("name"), friendlyDomain)
		}
	}

	domainRecord := core.NewRecord(domainCollection)
	domainRecord.Set("domain", friendlyDomain)
	domainRecord.Set("service", serviceRecord.Id)
	domainRecord.Set("use_https", "yes")

	return m.app.Save(domainRecord)
}



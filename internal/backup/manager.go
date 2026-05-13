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

type Manager struct {
	app          *pocketbase.PocketBase
	dataDir      string
	serviceRepo  repositories.ServiceRepository
	commandsRepo repositories.CommandsRepository
	downloader   *download.DownloadUsecase
	zipper       *ziphelper.Zip
	unzipper     *unzip.Unzip
}

func NewManager(
	app *pocketbase.PocketBase,
	cfg configs.Config,
	serviceRepo repositories.ServiceRepository,
	commandsRepo repositories.CommandsRepository,
	downloader *download.DownloadUsecase,
	zipper *ziphelper.Zip,
	unzipper *unzip.Unzip,
) *Manager {
	return &Manager{
		app:          app,
		dataDir:      cfg.GetDataDir(),
		serviceRepo:  serviceRepo,
		commandsRepo: commandsRepo,
		downloader:   downloader,
		zipper:       zipper,
		unzipper:     unzipper,
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

	serviceDir := filepath.Join(m.dataDir, service.ID)
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
		return nil, err
	}

	return &BackupFile{
		Path:     backupPath,
		Filename: fmt.Sprintf("%s-%s-backup.zip", sanitizeFilename(service.Name), service.ID),
	}, nil
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

	serviceDir := filepath.Join(m.dataDir, record.Id)
	if err := copyDir(filepath.Join(tempDir, "data"), serviceDir); err != nil {
		_ = m.app.Delete(record)
		_ = os.RemoveAll(serviceDir)
		return "", err
	}

	record.Set("status", string(models.Stopped))
	if err := m.app.Save(record); err != nil {
		return "", err
	}
	if err := m.commandsRepo.PublishStartCommand(ctx, record.Id); err != nil {
		return "", err
	}

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

	sourceDir := filepath.Join(m.dataDir, source.ID)
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

	targetDir := filepath.Join(m.dataDir, record.Id)
	if err := copyDir(sourceDir, targetDir); err != nil {
		_ = m.app.Delete(record)
		_ = os.RemoveAll(targetDir)
		return "", err
	}

	record.Set("status", string(models.Stopped))
	if err := m.app.Save(record); err != nil {
		return "", err
	}
	if err := m.commandsRepo.PublishStartCommand(ctx, record.Id); err != nil {
		return "", err
	}

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

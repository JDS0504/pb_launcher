package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
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
	Reader   io.ReadCloser
	Filename string
}

// SnapshotInfo representa un snapshot almacenado en la colección service_snapshots.
type SnapshotInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Comment   string    `json:"comment"`
	ServiceID string    `json:"service_id"`
	Type      string    `json:"type"` // "manual" | "pre-restore"
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Size      int64     `json:"size"`
	File      string    `json:"file"` // Nombre del archivo asignado por PocketBase
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

func (m *Manager) snapshotsDir(serviceID string) string {
	return filepath.Join(m.dataDir, "_snapshots", serviceID)
}

// zipRelPath devuelve la ruta relativa al dataDir del ZIP para un snapshot.
func (m *Manager) zipRelPath(serviceID, snapshotID string) string {
	return filepath.Join("_snapshots", serviceID, snapshotID+".zip")
}

// zipFullPath devuelve la ruta absoluta del ZIP dado su rel path.
func (m *Manager) zipFullPath(relPath string) string {
	return filepath.Join(m.dataDir, relPath)
}

// ListSnapshots lista todos los snapshots del servicio desde la BD, ordenados por fecha desc.
func (m *Manager) ListSnapshots(ctx context.Context, serviceID string) ([]SnapshotInfo, error) {
	if _, err := m.serviceRepo.FindService(ctx, serviceID); err != nil {
		return nil, err
	}

	records, err := m.app.FindAllRecords(
		collections.ServiceSnapshots,
		dbx.NewExp("service = {:id}", dbx.Params{"id": serviceID}),
	)
	if err != nil {
		return nil, err
	}

	snapshots := make([]SnapshotInfo, 0, len(records))
	for _, r := range records {
		snapshots = append(snapshots, m.recordToInfo(r))
	}

	// Ordenar por fecha descendente (más reciente primero)
	for i := 0; i < len(snapshots)-1; i++ {
		for j := i + 1; j < len(snapshots); j++ {
			if snapshots[j].CreatedAt.After(snapshots[i].CreatedAt) {
				snapshots[i], snapshots[j] = snapshots[j], snapshots[i]
			}
		}
	}

	return snapshots, nil
}

// CreateSnapshot crea un ZIP de la instancia y registra el snapshot en BD.
func (m *Manager) CreateSnapshot(ctx context.Context, serviceID string, name string, comment string) (*SnapshotInfo, error) {
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

	manifest := Manifest{
		Format:    manifestFormat,
		CreatedAt: time.Now().UTC(),
		Service: ManifestService{
			Name:             service.Name,
			ReleaseID:        service.ReleaseID,
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

	// Crear ZIP temporal
	tempFile, err := os.CreateTemp("", "pblauncher-snapshot-*.zip")
	if err != nil {
		return nil, err
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	if err := m.zipper.CreateFromDir(serviceDir, tempPath, "data", map[string][]byte{"manifest.json": manifestBytes}); err != nil {
		m.logger.Error(ctx, service.ID, "snapshot_create", err.Error(), nil)
		return nil, err
	}

	fileInfo, _ := os.Stat(tempPath)
	var size int64
	if fileInfo != nil {
		size = fileInfo.Size()
	}

	fileObj, err := filesystem.NewFileFromPath(tempPath)
	if err != nil {
		return nil, err
	}
	fileObj.Name = fmt.Sprintf("snapshot-%s-%d.zip", serviceID, time.Now().Unix())

	collection, err := m.app.FindCachedCollectionByNameOrId(collections.ServiceSnapshots)
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set("service", serviceID)
	record.Set("name", name)
	record.Set("comment", strings.TrimSpace(comment))
	record.Set("type", "manual")
	record.Set("version", service.Version)
	record.Set("file", fileObj)
	record.Set("size", size)

	if err := m.app.Save(record); err != nil {
		m.logger.Error(ctx, service.ID, "snapshot_create", err.Error(), nil)
		return nil, err
	}

	// Marcar que el estado actual del servicio coincide con este snapshot
	_ = m.setCurrentSnapshotID(service.ID, record.Id)

	info := m.recordToInfo(record)
	m.logger.Success(ctx, service.ID, "snapshot_create", "snapshot created successfully", map[string]any{"snapshot_id": record.Id, "name": name})
	return &info, nil
}

// RestoreSnapshotInPlace restaura un snapshot en el sitio, reemplazando los datos
// del servicio actual sin crear una nueva instancia.
// Antes de restaurar, crea un auto-snapshot "pre-restore" solo si el estado actual
// no tiene ya un snapshot asociado (evita duplicados innecesarios).
// Retorna el SnapshotInfo del auto-backup creado (o nil si se omitió).
func (m *Manager) RestoreSnapshotInPlace(ctx context.Context, serviceID string, snapshotID string) (*SnapshotInfo, error) {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	if service.Status != models.Stopped {
		return nil, fmt.Errorf("el servicio debe estar detenido para restaurar")
	}

	snapshotRecord, err := m.app.FindRecordById(collections.ServiceSnapshots, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found")
	}
	if snapshotRecord.GetString("service") != serviceID {
		return nil, fmt.Errorf("snapshot does not belong to this service")
	}

	// Auto-backup: solo si el estado actual está modificado (sucio) y el snapshot destino NO es un pre-restore
	isDirty := false
	if snapshotRecord.GetString("type") != "pre-restore" {
		serviceRecord, err := m.app.FindRecordById(collections.Services, serviceID)
		if err != nil {
			isDirty = true
		} else {
			currentSnapshotID := serviceRecord.GetString("current_snapshot_id")
			appliedTime := serviceRecord.GetDateTime("current_snapshot_applied_at").Time().UTC()

			if currentSnapshotID == "" || appliedTime.IsZero() {
				isDirty = true
			} else {
				dbPath := filepath.Join(m.dataDir, service.Name, "pb_data", "data.db")
				if fi, err := os.Stat(dbPath); err == nil {
					dbModTime := fi.ModTime().UTC()
					// Si el archivo de base de datos se modificó al menos 2 segundos después
					// de aplicar el último snapshot/restore, se considera sucio.
					if dbModTime.After(appliedTime.Add(2 * time.Second)) {
						isDirty = true
					}
				} else {
					isDirty = true
				}
			}
		}
	}

	var autoBackup *SnapshotInfo
	if isDirty {
		d := time.Now().UTC()
		autoName := "pre-restore-" + d.Format("2006-01-02-15-04")

		// Crear auto-backup marcado como "pre-restore"
		if ab, err := m.createSnapshotWithType(ctx, service, autoName, "Estado previo a restauración", "pre-restore"); err != nil {
			return nil, fmt.Errorf("no se pudo crear el auto-backup previo a la restauración: %w", err)
		} else {
			autoBackup = ab
		}
	}

	// Descargar el archivo ZIP de PocketBase (S3 o Local)
	tempZip, err := os.CreateTemp("", "pblauncher-restore-*.zip")
	if err != nil {
		return autoBackup, err
	}
	tempZipPath := tempZip.Name()
	defer os.Remove(tempZipPath)

	fs, err := m.app.NewFilesystem()
	if err != nil {
		tempZip.Close()
		return autoBackup, err
	}
	defer fs.Close()

	fileKey := snapshotRecord.BaseFilesPath() + "/" + snapshotRecord.GetString("file")
	reader, err := fs.GetFile(fileKey)
	if err != nil {
		tempZip.Close()
		return autoBackup, fmt.Errorf("no se pudo leer el archivo de snapshot desde el storage: %w", err)
	}
	defer reader.Close()

	if _, err := io.Copy(tempZip, reader); err != nil {
		tempZip.Close()
		return autoBackup, fmt.Errorf("error escribiendo el zip temporal: %w", err)
	}
	tempZip.Close()

	// Extraer el snapshot en un directorio temporal
	tempDir, err := os.MkdirTemp("", "pblauncher-inplace-restore-*")
	if err != nil {
		return autoBackup, err
	}
	defer os.RemoveAll(tempDir)

	if _, err := m.unzipper.Extract(tempZipPath, tempDir); err != nil {
		return autoBackup, fmt.Errorf("error extrayendo snapshot: %w", err)
	}

	manifest, err := readManifest(filepath.Join(tempDir, "manifest.json"))
	if err != nil {
		return autoBackup, err
	}
	if manifest.Format != manifestFormat {
		return autoBackup, fmt.Errorf("formato de backup no soportado: %q", manifest.Format)
	}

	// Reemplazar directorio del servicio
	serviceDir := filepath.Join(m.dataDir, service.Name)
	if err := os.RemoveAll(serviceDir); err != nil {
		return autoBackup, fmt.Errorf("no se pudo limpiar el directorio del servicio: %w", err)
	}
	if err := copyDir(filepath.Join(tempDir, "data"), serviceDir); err != nil {
		return autoBackup, fmt.Errorf("error restaurando datos (auto-backup disponible): %w", err)
	}

	// Marcar que el estado actual coincide con el snapshot restaurado
	_ = m.setCurrentSnapshotID(serviceID, snapshotID)

	m.logger.Success(ctx, serviceID, "snapshot_restore_inplace", "snapshot restaurado en sitio", map[string]any{
		"snapshot_id": snapshotID,
		"auto_backup": autoBackup != nil,
	})
	return autoBackup, nil
}

// RestoreSnapshot restaura un ZIP externo creando una nueva instancia (usado por upload de backup externo).
func (m *Manager) RestoreSnapshot(ctx context.Context, serviceID string, snapshotID string, name string) (string, error) {
	snapshotRecord, err := m.app.FindRecordById(collections.ServiceSnapshots, snapshotID)
	if err != nil {
		return "", fmt.Errorf("snapshot not found")
	}

	// Descargar el archivo ZIP de PocketBase (S3 o Local)
	tempZip, err := os.CreateTemp("", "pblauncher-restore-*.zip")
	if err != nil {
		return "", err
	}
	tempZipPath := tempZip.Name()
	defer os.Remove(tempZipPath)

	fs, err := m.app.NewFilesystem()
	if err != nil {
		tempZip.Close()
		return "", err
	}
	defer fs.Close()

	fileKey := snapshotRecord.BaseFilesPath() + "/" + snapshotRecord.GetString("file")
	reader, err := fs.GetFile(fileKey)
	if err != nil {
		tempZip.Close()
		return "", fmt.Errorf("no se pudo leer el archivo de snapshot desde el storage: %w", err)
	}
	defer reader.Close()

	if _, err := io.Copy(tempZip, reader); err != nil {
		tempZip.Close()
		return "", err
	}
	tempZip.Close()

	restoredID, err := m.Restore(ctx, tempZipPath, name)
	if err != nil {
		return "", err
	}
	m.logger.Success(ctx, restoredID, "snapshot_restore", "snapshot restored as new instance", map[string]any{"snapshot_id": snapshotID, "source_service_id": serviceID})
	return restoredID, nil
}

// DeleteSnapshot elimina un snapshot de la BD y automáticamente PocketBase borra el archivo de S3/Local.
func (m *Manager) DeleteSnapshot(ctx context.Context, serviceID string, snapshotID string) error {
	record, err := m.app.FindRecordById(collections.ServiceSnapshots, snapshotID)
	if err != nil {
		return fmt.Errorf("snapshot not found")
	}
	if record.GetString("service") != serviceID {
		return fmt.Errorf("snapshot does not belong to this service")
	}

	// Al eliminar el registro, PocketBase elimina automáticamente los archivos asociados de S3 o Local.
	if err := m.app.Delete(record); err != nil {
		return err
	}

	// Si este era el snapshot actual del servicio, limpiar la referencia
	if m.getCurrentSnapshotID(serviceID) == snapshotID {
		_ = m.setCurrentSnapshotID(serviceID, "")
	}

	m.logger.Success(ctx, serviceID, "snapshot_delete", "snapshot deleted", map[string]any{"snapshot_id": snapshotID})
	return nil
}

type wrappedCloser struct {
	reader io.Closer
	fs     io.Closer
}

func (w wrappedCloser) Close() error {
	_ = w.reader.Close()
	return w.fs.Close()
}

// GetSnapshotFile retorna la info del archivo ZIP para descarga.
func (m *Manager) GetSnapshotFile(ctx context.Context, serviceID string, snapshotID string) (*BackupFile, error) {
	record, err := m.app.FindRecordById(collections.ServiceSnapshots, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found")
	}
	if record.GetString("service") != serviceID {
		return nil, fmt.Errorf("snapshot does not belong to this service")
	}

	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	fs, err := m.app.NewFilesystem()
	if err != nil {
		return nil, err
	}

	fileKey := record.BaseFilesPath() + "/" + record.GetString("file")
	reader, err := fs.GetFile(fileKey)
	if err != nil {
		fs.Close()
		return nil, fmt.Errorf("no se pudo leer el archivo desde el storage: %w", err)
	}

	wrappedReader := &struct {
		io.Reader
		io.Closer
	}{
		Reader: reader,
		Closer: wrappedCloser{reader, fs},
	}

	return &BackupFile{
		Reader:   wrappedReader,
		Filename: fmt.Sprintf("%s-%s-snapshot.zip", sanitizeFilename(service.Name), snapshotID),
	}, nil
}

// findSnapshotRecord busca un snapshot en BD validando que pertenezca al servicio indicado.
func (m *Manager) findSnapshotRecord(ctx context.Context, serviceID string, snapshotID string) (*SnapshotInfo, error) {
	snapshotID = strings.TrimSpace(snapshotID)
	if snapshotID == "" || strings.Contains(snapshotID, "/") || strings.Contains(snapshotID, "..") {
		return nil, fmt.Errorf("invalid snapshot id")
	}

	record, err := m.app.FindRecordById(collections.ServiceSnapshots, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found")
	}
	if record.GetString("service") != serviceID {
		return nil, fmt.Errorf("snapshot does not belong to this service")
	}

	info := m.recordToInfo(record)
	return &info, nil
}

// recordToInfo convierte un registro PB a SnapshotInfo.
func (m *Manager) recordToInfo(r *core.Record) SnapshotInfo {
	return SnapshotInfo{
		ID:        r.Id,
		Name:      r.GetString("name"),
		Comment:   r.GetString("comment"),
		ServiceID: r.GetString("service"),
		Type:      r.GetString("type"),
		Version:   r.GetString("version"),
		CreatedAt: r.GetDateTime("created").Time(),
		Size:      int64(r.GetInt("size")),
		File:      r.GetString("file"),
	}
}

// getCurrentSnapshotID obtiene el current_snapshot_id del servicio desde la BD.
func (m *Manager) getCurrentSnapshotID(serviceID string) string {
	record, err := m.app.FindRecordById(collections.Services, serviceID)
	if err != nil {
		return ""
	}
	return record.GetString("current_snapshot_id")
}

// setCurrentSnapshotID actualiza el current_snapshot_id del servicio y su timestamp de aplicación.
func (m *Manager) setCurrentSnapshotID(serviceID, snapshotID string) error {
	record, err := m.app.FindRecordById(collections.Services, serviceID)
	if err != nil {
		return err
	}
	record.Set("current_snapshot_id", snapshotID)
	if snapshotID == "" {
		record.Set("current_snapshot_applied_at", nil)
	} else {
		record.Set("current_snapshot_applied_at", time.Now().UTC())
	}
	return m.app.Save(record)
}

// createSnapshotWithType es un helper interno para crear snapshots con tipo específico.
func (m *Manager) createSnapshotWithType(ctx context.Context, service *models.Service, name, comment, snapshotType string) (*SnapshotInfo, error) {
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

	// Crear ZIP temporal
	tempFile, err := os.CreateTemp("", "pblauncher-snapshot-*.zip")
	if err != nil {
		return nil, err
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	if err := m.zipper.CreateFromDir(serviceDir, tempPath, "data", map[string][]byte{"manifest.json": manifestBytes}); err != nil {
		return nil, err
	}

	fileInfo, _ := os.Stat(tempPath)
	var size int64
	if fileInfo != nil {
		size = fileInfo.Size()
	}

	fileObj, err := filesystem.NewFileFromPath(tempPath)
	if err != nil {
		return nil, err
	}
	fileObj.Name = fmt.Sprintf("snapshot-%s-%d.zip", service.ID, time.Now().Unix())

	collection, err := m.app.FindCachedCollectionByNameOrId(collections.ServiceSnapshots)
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set("service", service.ID)
	record.Set("name", name)
	record.Set("comment", comment)
	record.Set("type", snapshotType)
	record.Set("version", service.Version)
	record.Set("file", fileObj)
	record.Set("size", size)

	if err := m.app.Save(record); err != nil {
		return nil, err
	}

	_ = m.setCurrentSnapshotID(service.ID, record.Id)
	info := m.recordToInfo(record)
	return &info, nil
}


// Restore restaura un ZIP externo creando una nueva instancia PocketBase.
func (m *Manager) Restore(ctx context.Context, backupPath string, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("instance name is required")
	}

	// Validar que el nombre no esté duplicado en la base de datos
	existing, err := m.app.FindFirstRecordByFilter(
		collections.Services,
		"name = {:name}",
		dbx.Params{"name": name},
	)
	if err == nil && existing != nil {
		return "", fmt.Errorf("ya existe una instancia con el nombre '%s'", name)
	}

	// Validar que la carpeta de datos física no exista en disco
	checkDir := filepath.Join(m.dataDir, name)
	if _, statErr := os.Stat(checkDir); statErr == nil {
		return "", fmt.Errorf("ya existe un directorio de datos para '%s', elige un nombre diferente", name)
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

	releaseID := manifest.Service.ReleaseID
	release, err := m.serviceRepo.FindRelease(ctx, releaseID)
	if err != nil {
		fallback, ferr := m.app.FindFirstRecordByFilter(
			collections.Releases,
			"version = {:version}",
			map[string]any{"version": manifest.Service.Version},
		)
		if ferr != nil {
			return "", fmt.Errorf("backup release not found (id=%s, version=%s): %w", releaseID, manifest.Service.Version, err)
		}
		releaseID = fallback.Id
		release = &models.Release{ID: fallback.Id, Version: fallback.GetString("version")}
	}
	if release.Version != manifest.Service.Version {
		return "", fmt.Errorf("backup release version mismatch: expected %q, got %q", manifest.Service.Version, release.Version)
	}
	if err := m.downloader.EnsureReleaseDownloaded(ctx, releaseID); err != nil {
		return "", fmt.Errorf("failed to download backup release: %w", err)
	}

	collection, err := m.app.FindCachedCollectionByNameOrId(collections.Services)
	if err != nil {
		return "", err
	}
	record := core.NewRecord(collection)
	record.Set("name", name)
	record.Set("release", releaseID)
	record.Set("restart_policy", manifest.Service.RestartPolicy)
	record.Set("status", string(models.Restoring))
	record.Set("boot_user_email", manifest.Service.BootUserEmail)
	record.Set("boot_user_password", manifest.Service.BootUserPassword)

	if err := m.app.Save(record); err != nil {
		return "", err
	}

	serviceDir := filepath.Join(m.dataDir, record.GetString("name"))
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

// Clone clona una instancia en una nueva.
func (m *Manager) Clone(ctx context.Context, sourceServiceID string, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("instance name is required")
	}

	// Validar que el nombre no esté duplicado en la base de datos
	existing, err := m.app.FindFirstRecordByFilter(
		collections.Services,
		"name = {:name}",
		dbx.Params{"name": name},
	)
	if err == nil && existing != nil {
		return "", fmt.Errorf("ya existe una instancia con el nombre '%s'", name)
	}

	// Validar que la carpeta de datos física no exista en disco
	checkDir := filepath.Join(m.dataDir, name)
	if _, statErr := os.Stat(checkDir); statErr == nil {
		return "", fmt.Errorf("ya existe un directorio de datos para '%s', elige un nombre diferente", name)
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
	record.Set("cpu_quota", source.CpuQuota)
	record.Set("memory_limit", source.MemoryLimit)

	if err := m.app.Save(record); err != nil {
		return "", err
	}

	targetDir := filepath.Join(m.dataDir, record.GetString("name"))
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
	return CreateFriendlyDomain(m.app, serviceRecord, m.domainBase)
}

// logWarn es un helper para loguear advertencias sin afectar el flujo.
func logWarn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

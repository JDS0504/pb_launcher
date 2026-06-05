package filemanager

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"pb_launcher/configs"
	"pb_launcher/helpers/unzip"
	"pb_launcher/internal/launcher/domain/models"
	"pb_launcher/internal/launcher/domain/repositories"
	"pb_launcher/internal/operationlog"
	"strings"
	"time"
)

// compressibleForGzip contiene las extensiones que vale la pena comprimir en pb_public.
// Las imágenes (png, jpg, webp) ya están comprimidas internamente y no se benefician.
var compressibleForGzip = map[string]bool{
	".html": true,
	".js":   true,
	".css":  true,
	".json": true,
	".svg":  true,
	".xml":  true,
	".txt":  true,
	".wasm": true,
	".map":  true,
}

// isPbPublicPath informa si una ruta relativa pertenece a pb_public.
func isPbPublicPath(relPath string) bool {
	return relPath == "pb_public" || strings.HasPrefix(relPath, "pb_public"+string(os.PathSeparator))
}

// compressToGzip genera <fullPath>.gz con nivel de compresión máximo (BestCompression).
// Si la extensión no es comprimible, retorna nil sin hacer nada.
// La operación es one-shot: se realiza una vez al guardar, sin costo en cada request.
func compressToGzip(fullPath string) error {
	ext := strings.ToLower(filepath.Ext(fullPath))
	if !compressibleForGzip[ext] {
		return nil
	}

	src, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(fullPath + ".gz")
	if err != nil {
		return err
	}
	defer dst.Close()

	gz, err := gzip.NewWriterLevel(dst, gzip.BestCompression)
	if err != nil {
		return err
	}

	if _, err := io.Copy(gz, src); err != nil {
		gz.Close()
		return err
	}

	// gz.Close() escribe el footer de gzip — es obligatorio llamarlo explícitamente.
	return gz.Close()
}

type FileEntry struct {
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
	IsDir     bool      `json:"is_dir"`
}

type FileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type Manager struct {
	dataDir     string
	serviceRepo repositories.ServiceRepository
	logger      *operationlog.Logger
}

func NewManager(
	cfg configs.Config,
	serviceRepo repositories.ServiceRepository,
	logger *operationlog.Logger,
) *Manager {
	return &Manager{
		dataDir:     cfg.GetDataDir(),
		serviceRepo: serviceRepo,
		logger:      logger,
	}
}

func (m *Manager) serviceDir(serviceID string) string {
	return filepath.Join(m.dataDir, serviceID)
}

func (m *Manager) List(ctx context.Context, serviceID string) ([]FileEntry, error) {
	if _, err := m.serviceRepo.FindService(ctx, serviceID); err != nil {
		return nil, err
	}

	baseDir := m.serviceDir(serviceID)
	if _, err := os.Stat(baseDir); err != nil {
		if os.IsNotExist(err) {
			return []FileEntry{}, nil
		}
		return nil, err
	}

	allowedDirs := []string{"pb_hooks", "pb_public", "pb_migrations", "pb_data"}
	files := []FileEntry{}

	for _, sub := range allowedDirs {
		targetDir := filepath.Join(baseDir, sub)
		if _, err := os.Stat(targetDir); os.IsNotExist(err) {
			_ = os.MkdirAll(targetDir, 0755)
		}

		err := filepath.WalkDir(targetDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			relPath, err := safeRel(baseDir, path)
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			files = append(files, FileEntry{
				Path:      filepath.ToSlash(relPath),
				Size:      info.Size(),
				UpdatedAt: info.ModTime(),
				IsDir:     d.IsDir(),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func (m *Manager) ReadFile(ctx context.Context, serviceID string, targetPath string) (*FileContent, error) {
	if _, err := m.serviceRepo.FindService(ctx, serviceID); err != nil {
		return nil, err
	}
	relPath, err := validateFilePath(targetPath)
	if err != nil {
		return nil, err
	}
	fullPath, err := safeJoin(m.serviceDir(serviceID), relPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}
	// Límite de tamaño para proteger la UI (máximo 5MB para editar en web)
	if info.Size() > 5*1024*1024 {
		return nil, fmt.Errorf("el archivo excede el tamaño máximo permitido para edición web (5MB)")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	return &FileContent{Path: filepath.ToSlash(relPath), Content: string(data)}, nil
}

func (m *Manager) SaveFileBytes(ctx context.Context, serviceID string, targetPath string, data []byte) error {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return err
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("el servicio debe estar detenido para poder modificar archivos")
	}
	relPath, err := validateFilePath(targetPath)
	if err != nil {
		return err
	}
	fullPath, err := safeJoin(m.serviceDir(serviceID), relPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		m.logger.Error(ctx, service.ID, "file_save", err.Error(), map[string]any{"path": filepath.ToSlash(relPath)})
		return err
	}

	// Comprimir en pb_public una sola vez al guardar (gzip nivel 9, stdlib).
	if isPbPublicPath(relPath) {
		if gzErr := compressToGzip(fullPath); gzErr != nil {
			slog.Warn("filemanager: no se pudo comprimir archivo", "path", fullPath, "error", gzErr)
		}
	}

	m.logger.Success(ctx, service.ID, "file_save", "Archivo guardado exitosamente", map[string]any{"path": filepath.ToSlash(relPath)})
	return nil
}

func (m *Manager) SaveFile(ctx context.Context, serviceID string, targetPath string, content string) error {
	return m.SaveFileBytes(ctx, serviceID, targetPath, []byte(content))
}

// SaveFileStream escribe el contenido de src directamente al disco mediante streaming (io.Copy),
// sin cargar el archivo completo en RAM. Ideal para uploads de archivos grandes.
func (m *Manager) SaveFileStream(ctx context.Context, serviceID string, targetPath string, src io.Reader) error {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return err
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("el servicio debe estar detenido para poder modificar archivos")
	}
	relPath, err := validateFilePath(targetPath)
	if err != nil {
		return err
	}
	fullPath, err := safeJoin(m.serviceDir(serviceID), relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	dst, err := os.Create(fullPath)
	if err != nil {
		m.logger.Error(ctx, service.ID, "file_save", err.Error(), map[string]any{"path": filepath.ToSlash(relPath)})
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		m.logger.Error(ctx, service.ID, "file_save", err.Error(), map[string]any{"path": filepath.ToSlash(relPath)})
		return err
	}
	// Cierre explícito antes de comprimir: el archivo debe estar completamente escrito en disco.
	dst.Close()

	// Comprimir en pb_public una sola vez al guardar (gzip nivel 9, stdlib).
	if isPbPublicPath(relPath) {
		if gzErr := compressToGzip(fullPath); gzErr != nil {
			slog.Warn("filemanager: no se pudo comprimir archivo", "path", fullPath, "error", gzErr)
		}
	}

	m.logger.Success(ctx, service.ID, "file_save", "Archivo guardado exitosamente", map[string]any{"path": filepath.ToSlash(relPath)})
	return nil
}

func (m *Manager) DeleteFile(ctx context.Context, serviceID string, targetPath string) error {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return err
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("el servicio debe estar detenido para poder eliminar archivos")
	}
	relPath, err := validateFilePath(targetPath)
	if err != nil {
		return err
	}
	fullPath, err := safeJoin(m.serviceDir(serviceID), relPath)
	if err != nil {
		return err
	}
	// Si el archivo o directorio ya no existe, consideramos que la eliminación fue exitosa
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil
	}

	if err := os.RemoveAll(fullPath); err != nil {
		m.logger.Error(ctx, service.ID, "file_delete", err.Error(), map[string]any{"path": filepath.ToSlash(relPath)})
		return err
	}
	// Eliminar el .gz generado automáticamente si existe.
	_ = os.Remove(fullPath + ".gz")
	cleanEmptyParents(m.serviceDir(serviceID), filepath.Dir(fullPath))
	m.logger.Success(ctx, service.ID, "file_delete", "Archivo eliminado exitosamente", map[string]any{"path": filepath.ToSlash(relPath)})
	return nil
}

func safeRel(base, target string) (string, error) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	relPath, err := filepath.Rel(baseAbs, targetAbs)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("ruta de archivo ilegal detectada: %s", target)
	}
	return relPath, nil
}

func validateFilePath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", fmt.Errorf("la ruta del archivo es obligatoria")
	}
	if filepath.IsAbs(rawPath) {
		return "", fmt.Errorf("no se permiten rutas absolutas")
	}
	cleanPath := filepath.Clean(filepath.FromSlash(rawPath))
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("ruta ilegal")
	}

	// Comprobar que comience con alguna de las carpetas permitidas
	allowed := []string{"pb_hooks", "pb_public", "pb_migrations", "pb_data"}
	valid := false
	for _, dir := range allowed {
		if cleanPath == dir || strings.HasPrefix(cleanPath, dir+string(os.PathSeparator)) {
			valid = true
			break
		}
	}
	if !valid {
		return "", fmt.Errorf("acceso restringido: el archivo debe estar dentro de pb_hooks, pb_data, pb_public o pb_migrations")
	}

	return cleanPath, nil
}

func safeJoin(base string, relPath string) (string, error) {
	target := filepath.Join(base, relPath)
	if _, err := safeRel(base, target); err != nil {
		return "", err
	}
	return target, nil
}

func cleanEmptyParents(base string, dir string) {
	baseAbs, err := filepath.Abs(base)
	if err != nil {
		return
	}
	for {
		dirAbs, err := filepath.Abs(dir)
		if err != nil || dirAbs == baseAbs {
			return
		}
		rel, err := filepath.Rel(baseAbs, dirAbs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return
		}

		// Validamos que sea un subdirectorio válido de los permitidos
		relToSlash := filepath.ToSlash(rel)
		validSub := false
		for _, allowedSub := range []string{"pb_hooks", "pb_public", "pb_migrations", "pb_data"} {
			if relToSlash == allowedSub || strings.HasPrefix(relToSlash, allowedSub+"/") {
				validSub = true
				break
			}
		}
		if !validSub {
			return
		}

		if err := os.Remove(dirAbs); err != nil {
			return
		}
		dir = filepath.Dir(dirAbs)
	}
}

func (m *Manager) GetSafeFilePath(ctx context.Context, serviceID string, targetPath string) (string, error) {
	if _, err := m.serviceRepo.FindService(ctx, serviceID); err != nil {
		return "", err
	}
	relPath, err := validateFilePath(targetPath)
	if err != nil {
		return "", err
	}
	return safeJoin(m.serviceDir(serviceID), relPath)
}

func (m *Manager) CreateDirectory(ctx context.Context, serviceID string, targetPath string) error {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return err
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("el servicio debe estar detenido para poder crear carpetas")
	}
	relPath, err := validateFilePath(targetPath)
	if err != nil {
		return err
	}
	fullPath, err := safeJoin(m.serviceDir(serviceID), relPath)
	if err != nil {
		return err
	}
	return os.MkdirAll(fullPath, 0755)
}

func (m *Manager) RenameFile(ctx context.Context, serviceID string, oldPath string, newPath string) error {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return err
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("el servicio debe estar detenido para poder mover/renombrar archivos")
	}
	relOld, err := validateFilePath(oldPath)
	if err != nil {
		return err
	}
	relNew, err := validateFilePath(newPath)
	if err != nil {
		return err
	}
	fullOld, err := safeJoin(m.serviceDir(serviceID), relOld)
	if err != nil {
		return err
	}
	fullNew, err := safeJoin(m.serviceDir(serviceID), relNew)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(fullNew), 0755); err != nil {
		return err
	}
	if err := os.Rename(fullOld, fullNew); err != nil {
		return err
	}

	cleanEmptyParents(m.serviceDir(serviceID), filepath.Dir(fullOld))
	return nil
}

func (m *Manager) ExtractZip(ctx context.Context, serviceID string, zipPath string) error {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return err
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("el servicio debe estar detenido para poder extraer archivos ZIP")
	}
	relPath, err := validateFilePath(zipPath)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(strings.ToLower(relPath), ".zip") {
		return fmt.Errorf("el archivo seleccionado no es un ZIP válido")
	}

	fullZipPath, err := safeJoin(m.serviceDir(serviceID), relPath)
	if err != nil {
		return err
	}

	destDir := filepath.Dir(fullZipPath)
	uz := unzip.NewUnzip()
	_, err = uz.Extract(fullZipPath, destDir)
	if err != nil {
		m.logger.Error(ctx, service.ID, "file_unzip", err.Error(), map[string]any{"path": filepath.ToSlash(relPath)})
		return err
	}

	m.logger.Success(ctx, service.ID, "file_unzip", "Archivo ZIP extraído exitosamente", map[string]any{"path": filepath.ToSlash(relPath)})
	return nil
}

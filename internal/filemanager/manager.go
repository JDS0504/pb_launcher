package filemanager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"pb_launcher/configs"
	"pb_launcher/internal/launcher/domain/models"
	"pb_launcher/internal/launcher/domain/repositories"
	"pb_launcher/internal/operationlog"
	"strings"
	"time"
)

type FileEntry struct {
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
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
		if _, err := os.Stat(targetDir); err != nil {
			continue // Si el subdirectorio no existe, simplemente pasamos
		}

		err := filepath.WalkDir(targetDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
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

func (m *Manager) SaveFile(ctx context.Context, serviceID string, targetPath string, content string) error {
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
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		m.logger.Error(ctx, service.ID, "file_save", err.Error(), map[string]any{"path": filepath.ToSlash(relPath)})
		return err
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

	if err := os.Remove(fullPath); err != nil {
		m.logger.Error(ctx, service.ID, "file_delete", err.Error(), map[string]any{"path": filepath.ToSlash(relPath)})
		return err
	}
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

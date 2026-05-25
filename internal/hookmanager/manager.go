package hookmanager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"pb_launcher/configs"
	"pb_launcher/helpers/unzip"
	ziphelper "pb_launcher/helpers/zip"
	"pb_launcher/internal/launcher/domain/models"
	"pb_launcher/internal/launcher/domain/repositories"
	"pb_launcher/internal/operationlog"
	"strings"
	"time"
)

type HookFile struct {
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HookFileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ExportFile struct {
	Path     string
	Filename string
}

type Manager struct {
	dataDir     string
	serviceRepo repositories.ServiceRepository
	zipper      *ziphelper.Zip
	unzipper    *unzip.Unzip
	logger      *operationlog.Logger
}

func NewManager(
	cfg configs.Config,
	serviceRepo repositories.ServiceRepository,
	zipper *ziphelper.Zip,
	unzipper *unzip.Unzip,
	logger *operationlog.Logger,
) *Manager {
	return &Manager{
		dataDir:     cfg.GetDataDir(),
		serviceRepo: serviceRepo,
		zipper:      zipper,
		unzipper:    unzipper,
		logger:      logger,
	}
}

func (m *Manager) hooksDir(serviceID string) string {
	return filepath.Join(m.dataDir, serviceID, "pb_hooks")
}

func (m *Manager) List(ctx context.Context, serviceID string) ([]HookFile, error) {
	if _, err := m.serviceRepo.FindService(ctx, serviceID); err != nil {
		return nil, err
	}

	hooksDir := m.hooksDir(serviceID)
	if _, err := os.Stat(hooksDir); err != nil {
		if os.IsNotExist(err) {
			return []HookFile{}, nil
		}
		return nil, err
	}

	files := []HookFile{}
	if err := filepath.WalkDir(hooksDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".pb.js") {
			return nil
		}
		relPath, err := safeRel(hooksDir, path)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files = append(files, HookFile{
			Path:      filepath.ToSlash(relPath),
			Size:      info.Size(),
			UpdatedAt: info.ModTime(),
		})
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func (m *Manager) Export(ctx context.Context, serviceID string) (*ExportFile, error) {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	sourceDir := m.hooksDir(serviceID)
	if _, err := os.Stat(sourceDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		emptyDir, err := os.MkdirTemp("", "pblauncher-hooks-empty-*")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(emptyDir)
		sourceDir = emptyDir
	}

	exportPath := filepath.Join(os.TempDir(), fmt.Sprintf("pblauncher-hooks-%s-%d.zip", service.ID, time.Now().Unix()))
	if err := m.zipper.CreateFromDir(sourceDir, exportPath, "", nil); err != nil {
		m.logger.Error(ctx, service.ID, "hooks_export", err.Error(), nil)
		return nil, err
	}
	m.logger.Success(ctx, service.ID, "hooks_export", "PB hooks exported successfully", nil)

	return &ExportFile{
		Path:     exportPath,
		Filename: fmt.Sprintf("%s-hooks.zip", sanitizeFilename(service.Name)),
	}, nil
}

func (m *Manager) ReadFile(ctx context.Context, serviceID string, hookPath string) (*HookFileContent, error) {
	if _, err := m.serviceRepo.FindService(ctx, serviceID); err != nil {
		return nil, err
	}
	relPath, err := validateHookPath(hookPath)
	if err != nil {
		return nil, err
	}
	fullPath, err := safeJoin(m.hooksDir(serviceID), relPath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	return &HookFileContent{Path: filepath.ToSlash(relPath), Content: string(data)}, nil
}

func (m *Manager) SaveFile(ctx context.Context, serviceID string, hookPath string, content string) error {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return err
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("service must be stopped before modifying PB hooks")
	}
	relPath, err := validateHookPath(hookPath)
	if err != nil {
		return err
	}
	fullPath, err := safeJoin(m.hooksDir(serviceID), relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		m.logger.Error(ctx, service.ID, "hook_save", err.Error(), map[string]any{"path": filepath.ToSlash(relPath)})
		return err
	}
	m.logger.Success(ctx, service.ID, "hook_save", "PB hook saved successfully", map[string]any{"path": filepath.ToSlash(relPath)})
	return nil
}

func (m *Manager) DeleteFile(ctx context.Context, serviceID string, hookPath string) error {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return err
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("service must be stopped before deleting PB hooks")
	}
	relPath, err := validateHookPath(hookPath)
	if err != nil {
		return err
	}
	fullPath, err := safeJoin(m.hooksDir(serviceID), relPath)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil {
		m.logger.Error(ctx, service.ID, "hook_delete", err.Error(), map[string]any{"path": filepath.ToSlash(relPath)})
		return err
	}
	cleanEmptyParents(m.hooksDir(serviceID), filepath.Dir(fullPath))
	m.logger.Success(ctx, service.ID, "hook_delete", "PB hook deleted successfully", map[string]any{"path": filepath.ToSlash(relPath)})
	return nil
}

func (m *Manager) Import(ctx context.Context, serviceID string, zipPath string) ([]string, error) {
	service, err := m.serviceRepo.FindService(ctx, serviceID)
	if err != nil {
		return nil, err
	}
	if service.Status != models.Stopped {
		return nil, fmt.Errorf("service must be stopped before importing PB hooks")
	}

	tempDir, err := os.MkdirTemp("", "pblauncher-hooks-import-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	if _, err := m.unzipper.Extract(zipPath, tempDir); err != nil {
		m.logger.Error(ctx, service.ID, "hooks_import", err.Error(), nil)
		return nil, err
	}

	root, hookFiles, err := detectHookRoot(tempDir)
	if err != nil {
		m.logger.Error(ctx, service.ID, "hooks_import", err.Error(), nil)
		return nil, err
	}
	if err := validateOnlyPBJS(root); err != nil {
		m.logger.Error(ctx, service.ID, "hooks_import", err.Error(), nil)
		return nil, err
	}

	stagedDir, err := os.MkdirTemp("", "pblauncher-hooks-stage-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(stagedDir)

	imported := make([]string, 0, len(hookFiles))
	for _, sourcePath := range hookFiles {
		relPath, err := safeRel(root, sourcePath)
		if err != nil {
			return nil, err
		}
		if err := copyFile(sourcePath, filepath.Join(stagedDir, relPath)); err != nil {
			return nil, err
		}
		imported = append(imported, filepath.ToSlash(relPath))
	}

	targetDir := m.hooksDir(service.ID)
	if err := os.RemoveAll(targetDir); err != nil {
		return nil, err
	}
	if err := copyDir(stagedDir, targetDir); err != nil {
		return nil, err
	}

	m.logger.Success(ctx, service.ID, "hooks_import", "PB hooks imported successfully", map[string]any{"count": len(imported), "files": imported})
	return imported, nil
}

func detectHookRoot(extractedDir string) (string, []string, error) {
	hookFiles := []string{}
	if err := filepath.WalkDir(extractedDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".pb.js") {
			hookFiles = append(hookFiles, path)
		}
		return nil
	}); err != nil {
		return "", nil, err
	}
	if len(hookFiles) == 0 {
		return "", nil, fmt.Errorf("zip does not contain any .pb.js files")
	}

	root := filepath.Dir(hookFiles[0])
	for _, path := range hookFiles[1:] {
		root = commonDir(root, filepath.Dir(path))
	}
	return root, hookFiles, nil
}

func validateOnlyPBJS(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".pb.js") {
			relPath, _ := filepath.Rel(root, path)
			return fmt.Errorf("unsupported file in hooks root: %s", filepath.ToSlash(relPath))
		}
		return nil
	})
}

func commonDir(a, b string) string {
	aParts := strings.Split(filepath.Clean(a), string(os.PathSeparator))
	bParts := strings.Split(filepath.Clean(b), string(os.PathSeparator))
	limit := min(len(aParts), len(bParts))
	common := []string{}
	for i := 0; i < limit; i++ {
		if aParts[i] != bParts[i] {
			break
		}
		common = append(common, aParts[i])
	}
	if len(common) == 0 {
		return string(os.PathSeparator)
	}
	if common[0] == "" {
		return string(os.PathSeparator) + filepath.Join(common[1:]...)
	}
	return filepath.Join(common...)
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
		return "", fmt.Errorf("illegal file path detected: %s", target)
	}
	return relPath, nil
}

func validateHookPath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", fmt.Errorf("hook path is required")
	}
	if filepath.IsAbs(rawPath) {
		return "", fmt.Errorf("absolute hook paths are not allowed")
	}
	cleanPath := filepath.Clean(filepath.FromSlash(rawPath))
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("illegal hook path")
	}
	if !strings.HasSuffix(cleanPath, ".pb.js") {
		return "", fmt.Errorf("hook file must end with .pb.js")
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
		if rel, err := filepath.Rel(baseAbs, dirAbs); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return
		}
		if err := os.Remove(dirAbs); err != nil {
			return
		}
		dir = filepath.Dir(dirAbs)
	}
}

func copyDir(source, destination string) error {
	return filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := safeRel(source, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destination, relPath)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		return copyFile(path, targetPath)
	})
}

func copyFile(source, destination string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0644)
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

package domain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"pb_launcher/configs"
	"pb_launcher/helpers/logstore"
	"pb_launcher/helpers/process"
	download "pb_launcher/internal/download/domain"
	"pb_launcher/internal/launcher/domain/models"
	"pb_launcher/internal/launcher/domain/repositories"
	"pb_launcher/internal/launcher/domain/services"
	"pb_launcher/internal/operationlog"
	"pb_launcher/utils/iouitls"
	"pb_launcher/utils/networktools"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	version "github.com/hashicorp/go-version"
)

type LauncherManager struct {
	rwMtx               sync.RWMutex
	dataDir             string
	ipAddress           string
	installTokenUsecase *CleanServiceInstallTokenUsecase
	repository          repositories.ServiceRepository
	comandsRepository   repositories.CommandsRepository
	finder              services.BinaryFinder
	downloader          *download.DownloadUsecase
	lstore              *logstore.ServiceLogDB
	operationLogger     *operationlog.Logger
	//
	processList     map[string]*process.Process
	errChan         chan process.ProcessErrorMessage
	cpuQuota        string
	memoryLimit     string
	// Auto-Sleep fields
	activityMap         map[string]time.Time
	checkTickerInterval time.Duration
	idleTimeout         time.Duration
	stopChan            chan struct{}
}

func NewLauncherManager(
	installTokenUsecase *CleanServiceInstallTokenUsecase,
	repository repositories.ServiceRepository,
	comandsRepository repositories.CommandsRepository,
	finder services.BinaryFinder,
	downloader *download.DownloadUsecase,
	lstore *logstore.ServiceLogDB,
	operationLogger *operationlog.Logger,
	c configs.Config,
) *LauncherManager {
	lm := &LauncherManager{
		installTokenUsecase: installTokenUsecase,
		repository:          repository,
		comandsRepository:   comandsRepository,
		finder:              finder,
		downloader:          downloader,
		lstore:              lstore,
		operationLogger:     operationLogger,
		dataDir:             c.GetDataDir(),
		ipAddress:           c.GetBindIPAddress(),
		processList:         make(map[string]*process.Process),
		errChan:             make(chan process.ProcessErrorMessage, 10),
		cpuQuota:            c.GetInstanceCpuQuota(),
		memoryLimit:         c.GetInstanceMemoryLimit(),
		activityMap:         make(map[string]time.Time),
		checkTickerInterval: c.GetAutoSleepCheckInterval(),
		idleTimeout:         c.GetAutoSleepIdleTimeout(),
		stopChan:            make(chan struct{}),
	}
	go lm.handleServiceErrors()
	go lm.startAutoSleepTicker()
	return lm
}

func (lm *LauncherManager) handleServiceErrors() {
	for serviceErr := range lm.errChan {
		ctx := context.Background()
		var errorMessage string
		if serviceErr.Error != nil {
			errorMessage = serviceErr.Error.Error()
		}

		if err := lm.repository.MarkServiceFailure(ctx, serviceErr.ID, errorMessage); err != nil {
			slog.Error("failed to update service status",
				"serviceID", serviceErr.ID,
				"error", err,
				"originalError", errorMessage,
			)
			continue
		}

		service, err := lm.repository.FindService(ctx, serviceErr.ID)
		if err != nil {
			slog.Error("failed to find service",
				"serviceID", serviceErr.ID,
				"error", err,
			)
			continue
		}

		if service.RestartPolicy != models.OnFailure {
			continue
		}

		if err := lm.comandsRepository.PublishStartComand(ctx, service.ID); err != nil {
			slog.Error("failed to publish restart command",
				"serviceID", service.ID,
				"error", err,
			)
		}
	}
}

func (lm *LauncherManager) buildArgs(serviceID string) ([]string, error) {
	pb_data := path.Join(lm.dataDir, serviceID)
	return []string{
		"--dir", path.Join(pb_data, "pb_data"),
		"--hooksDir", path.Join(pb_data, "pb_hooks"),
		"--publicDir", path.Join(pb_data, "pb_public"),
		"--migrationsDir", path.Join(pb_data, "pb_migrations"),
	}, nil
}

func (lm *LauncherManager) findOrDownloadBinary(ctx context.Context, service models.Service) (string, error) {
	executablePath, err := lm.finder.FindBinary(ctx, service.RepositoryID, service.Version, service.ExecFilePattern)
	if err == nil {
		return executablePath, nil
	}

	slog.Warn("binary not found locally, downloading release",
		"serviceID", service.ID,
		"releaseID", service.ReleaseID,
		"version", service.Version,
		"error", err,
	)
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, fmt.Sprintf("Binary v%s not found locally. Downloading release...", service.Version))

	if err := lm.downloader.EnsureReleaseDownloaded(ctx, service.ReleaseID); err != nil {
		lm.lstore.InsertLog(service.ID, logstore.StreamStderr, fmt.Sprintf("Failed to download binary v%s: %s", service.Version, err.Error()))
		return "", fmt.Errorf("failed to download release %s: %w", service.ReleaseID, err)
	}

	executablePath, err = lm.finder.FindBinary(ctx, service.RepositoryID, service.Version, service.ExecFilePattern)
	if err != nil {
		lm.lstore.InsertLog(service.ID, logstore.StreamStderr, fmt.Sprintf("Binary v%s downloaded, but executable was not found: %s", service.Version, err.Error()))
		return "", err
	}

	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, fmt.Sprintf("Binary v%s downloaded successfully", service.Version))
	return executablePath, nil
}

// initializeBootUser sets up the initial boot user for the service instance.
func (lm *LauncherManager) UpsertSuperuser(ctx context.Context, serviceID, email, password string) error {
	service, err := lm.repository.FindService(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("failed to find service %s: %w", serviceID, err)
	}

	binaryPath, err := lm.findOrDownloadBinary(ctx, *service)
	if err != nil {
		slog.Error("failed to find binary", "serviceID", service.ID, "error", err)
		return err
	}
	baseArgs, err := lm.buildArgs(service.ID)
	if err != nil {
		slog.Error("failed to build args", "serviceID", service.ID, "error", err)
		return err
	}
	args := append(baseArgs, "superuser", "upsert", email, password)
	cmd := exec.CommandContext(ctx, binaryPath, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to initialize boot user",
			"service", service.ID,
			"email", service.BootUserEmail,
			"output", string(output),
			"error", err,
		)
		return err
	}
	return lm.repository.UpdateSuperuser(ctx, serviceID, email, password)
}

var pbInstallURLRegex = regexp.MustCompile(`https?://[^/]+/_/#/pbinstal/([a-zA-Z0-9._\-]+)`)

func (lm *LauncherManager) buildStdoutHandler(serviceID string) iouitls.WriterInterceptorHandler {
	pbpbinstal := []byte("/pbinstal/")
	return func(data []byte) {
		if !bytes.Contains(data, pbpbinstal) {
			return
		}
		searchData := data
		if len(data) > 2048 {
			searchData = data[:2048]
		}
		matched := pbInstallURLRegex.FindSubmatch(searchData)
		if len(matched) < 2 {
			return
		}
		token := strings.TrimSpace(string(matched[1]))
		go lm.installTokenUsecase.SetInstallToken(context.Background(), serviceID, token)
	}
}

func (lm *LauncherManager) startService(ctx context.Context, service models.Service) error {
	lm.rwMtx.Lock()
	defer lm.rwMtx.Unlock()
	return lm.startServiceLocked(ctx, service)
}

func (lm *LauncherManager) startServiceLocked(ctx context.Context, service models.Service) error {
	if existingProcess, exists := lm.processList[service.ID]; exists {
		if existingProcess.IsRunning() {
			return fmt.Errorf("service %s is already running", service.ID)
		}
	}

	executablePath, err := lm.findOrDownloadBinary(ctx, service)
	if err != nil {
		slog.Error("failed to find binary", "serviceID", service.ID, "error", err)
		return err
	}

	serviceDir := path.Join(lm.dataDir, service.ID)
	ensureDir := func(name string) {
		p := path.Join(serviceDir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err := os.MkdirAll(p, 0755); err != nil {
				slog.Error("failed to create standard service directory", "path", p, "error", err)
			}
		}
	}
	ensureDir("pb_public")
	ensureDir("pb_hooks")
	ensureDir("pb_migrations")

	var ip string
	var port int

	useExistingPort := false
	if service.Port != "" {
		if p, errConv := strconv.Atoi(service.Port); errConv == nil && p > 0 {
			if networktools.IsPortAvailable(lm.ipAddress, p) {
				ip = lm.ipAddress
				port = p
				useExistingPort = true
				slog.Info("reusing existing persistent port for service", "serviceID", service.ID, "port", port)
			}
		}
	}

	if !useExistingPort {
		ip, port, err = networktools.GetAvailablePort(lm.ipAddress)
		if err != nil {
			slog.Error("failed to find free port", "serviceID", service.ID, "error", err)
			return err
		}
		slog.Info("assigned new dynamic port for service", "serviceID", service.ID, "port", port)
	}

	baseArgs, err := lm.buildArgs(service.ID)
	if err != nil {
		slog.Error("failed to build args", "serviceID", service.ID, "error", err)
		return err
	}

	listenIp := fmt.Sprintf("%s:%d", ip, port)
	serveArgs := append([]string{"serve"}, append(baseArgs, "--http", listenIp)...)

	stdout := iouitls.NewWriterInterceptor(
		lm.lstore.NewWriter(service.ID, logstore.StreamStdout),
		lm.buildStdoutHandler(service.ID),
	)

	newProcess := process.New(
		service.ID,
		executablePath,
		serveArgs,
		process.WithErrorChan(lm.errChan),
		process.WithStdout(stdout),
		process.WithStderr(lm.lstore.NewWriter(service.ID, logstore.StreamStderr)),
		process.WithCpuQuota(lm.cpuQuota),
		process.WithMemoryLimit(lm.memoryLimit),
	)

	if err := newProcess.Start(); err != nil {
		slog.Error("failed to start process", "serviceID", service.ID, "error", err)

		stopErr := fmt.Errorf("failed to start process: %w", err)
		if err := lm.repository.MarkServiceFailure(ctx, service.ID, stopErr.Error()); err != nil {
			slog.Error("failed to mark service as failed", "serviceID", service.ID, "error", err)
		}
		return err
	}

	lm.processList[service.ID] = newProcess
	lm.activityMap[service.ID] = time.Now()

	if err := lm.repository.MarkServiceRunning(ctx, service.ID, ip, fmt.Sprint(port)); err != nil {
		slog.Error("failed to update service status to running",
			"serviceID", service.ID,
			"ip", ip,
			"port", port,
			"error", err,
		)
	}

	return err
}

func (lm *LauncherManager) stopService(ctx context.Context, serviceID string) error {
	lm.rwMtx.Lock()
	defer lm.rwMtx.Unlock()
	return lm.stopServiceLocked(ctx, serviceID)
}

func (lm *LauncherManager) stopProcessOnlyLocked(serviceID string) error {
	existingProcess, exists := lm.processList[serviceID]
	if !exists {
		return fmt.Errorf("no running process found for service %s", serviceID)
	}
	if !existingProcess.IsRunning() {
		return fmt.Errorf("process for service %s is not currently running", serviceID)
	}

	if err := existingProcess.Stop(); err != nil {
		return err
	}

	delete(lm.processList, serviceID)
	delete(lm.activityMap, serviceID)
	return nil
}

func (lm *LauncherManager) stopServiceLocked(ctx context.Context, serviceID string) error {
	if err := lm.stopProcessOnlyLocked(serviceID); err != nil {
		slog.Error("failed to stop existing process", "serviceID", serviceID, "error", err)
		return err
	}

	if err := lm.repository.MarkServiceStoped(ctx, serviceID); err != nil {
		slog.Error("failed to mark service as stopped", "serviceID", serviceID, "error", err)
	}
	return nil
}

func (lm *LauncherManager) restartService(ctx context.Context, service models.Service) error {
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, "Restarting service...")

	if p, ok := lm.processList[service.ID]; ok && p.IsRunning() {
		if err := lm.stopService(ctx, service.ID); err != nil {
			slog.Error("restart failed: unable to stop service", "serviceID", service.ID, "error", err)
			return err
		}
	}
	if err := lm.startService(ctx, service); err != nil {
		slog.Error("restart failed: unable to start service", "serviceID", service.ID, "error", err)
		return err
	}
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, "Service restarted successfully")
	return nil
}

func (lm *LauncherManager) upgradeService(ctx context.Context, service models.Service, targetReleaseID string) error {
	if targetReleaseID == "" {
		return fmt.Errorf("target release is required for service upgrade")
	}
	if service.Status != models.Stopped {
		return fmt.Errorf("service %s must be stopped before upgrade", service.ID)
	}

	targetRelease, err := lm.repository.FindRelease(ctx, targetReleaseID)
	if err != nil {
		return fmt.Errorf("failed to find target release %s: %w", targetReleaseID, err)
	}
	if targetRelease.RepositoryID != service.RepositoryID {
		return fmt.Errorf("target release belongs to a different repository")
	}

	currentVersion, err := version.NewVersion(service.Version)
	if err != nil {
		return fmt.Errorf("invalid current service version %q: %w", service.Version, err)
	}
	targetVersion, err := version.NewVersion(targetRelease.Version)
	if err != nil {
		return fmt.Errorf("invalid target release version %q: %w", targetRelease.Version, err)
	}
	if !targetVersion.GreaterThan(currentVersion) {
		return fmt.Errorf("target version %s must be greater than current version %s", targetVersion.String(), currentVersion.String())
	}

	validationService := service
	validationService.ReleaseID = targetRelease.ID
	validationService.Version = targetRelease.Version
	if _, err := lm.findOrDownloadBinary(ctx, validationService); err != nil {
		return fmt.Errorf("target version binary not found: %w", err)
	}

	if err := lm.repository.UpdateServiceRelease(ctx, service.ID, targetRelease.ID); err != nil {
		return fmt.Errorf("failed to update service release: %w", err)
	}

	upgradedService := service
	upgradedService.ReleaseID = targetRelease.ID
	upgradedService.Version = targetRelease.Version
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, fmt.Sprintf("Upgrading service from v%s to v%s...", service.Version, targetRelease.Version))
	if err := lm.startService(ctx, upgradedService); err != nil {
		slog.Error("upgrade failed: unable to start service", "serviceID", service.ID, "error", err)
		return err
	}
	lm.lstore.InsertLog(service.ID, logstore.StreamStdout, "Service upgraded successfully")
	return nil
}

// RecoveryLastState restores and starts all services that were active
// before pb_launcher was shut down.
func (lm *LauncherManager) RecoveryLastState(ctx context.Context) error {
	lm.rwMtx.Lock()
	defer lm.rwMtx.Unlock()
	services, err := lm.repository.RunningServices(ctx)
	if err != nil {
		slog.Error("Failed to retrieve running services", "error", err)
		return err
	}

	for _, service := range services {
		if service.Deleted != "" {
			continue
		}
		if err := lm.startServiceLocked(ctx, service); err != nil {
			slog.Error("failed to start service",
				"serviceID", service.ID,
				"error", err,
			)

			continue
		}
	}

	return nil
}

func (lm *LauncherManager) evaluateCommand(ctx context.Context, cmd models.ServiceCommand) error {
	service, err := lm.repository.FindService(ctx, cmd.Service)
	if err != nil {
		return fmt.Errorf("failed to find service %s: %w", cmd.Service, err)
	}

	switch cmd.Action {
	case models.ActionStart:
		return lm.startService(ctx, *service)
	case models.ActionStop:
		_ = lm.stopService(ctx, service.ID)

		if service.Deleted != "" {
			serviceDir := path.Join(lm.dataDir, service.ID)
			if err := os.RemoveAll(serviceDir); err != nil {
				slog.Error("failed to remove service data directory on deletion", "serviceID", service.ID, "path", serviceDir, "error", err)
				return fmt.Errorf("failed to remove data directory: %w", err)
			}
			slog.Info("successfully removed service data directory on deletion", "serviceID", service.ID, "path", serviceDir)
		}
		return nil
	case models.ActionRestart:
		return lm.restartService(ctx, *service)
	case models.ActionUpgrade:
		return lm.upgradeService(ctx, *service, cmd.TargetRelease)
	default:
		return fmt.Errorf("unknown action %q for service %s", cmd.Action, cmd.Service)
	}
}

func (lm *LauncherManager) Run(ctx context.Context) error {
	comands, err := lm.comandsRepository.GetPendingCommands(ctx)
	if err != nil {
		slog.Error("failed to get pending commands", "error", err)
		return err
	}
	for _, c := range comands {
		if err := lm.evaluateCommand(ctx, c); err != nil {
			lm.operationLogger.Error(ctx, c.Service, c.Action.String(), err.Error(), map[string]any{"command_id": c.ID})
			if markErr := lm.comandsRepository.MarkCommandError(ctx, c.ID, err.Error()); markErr != nil {
				slog.Error("failed to mark command as error", "commandID", c.ID, "error", markErr)
			}
			continue
		}
		lm.operationLogger.Success(ctx, c.Service, c.Action.String(), "command executed successfully", map[string]any{"command_id": c.ID})
		if err := lm.comandsRepository.MarkCommandSuccess(ctx, c.ID); err != nil {
			slog.Error("failed to mark command as success", "commandID", c.ID, "error", err)
		}
	}
	return nil
}

func (lm *LauncherManager) Dispose() error {
	lm.rwMtx.Lock()
	defer lm.rwMtx.Unlock()

	close(lm.stopChan)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var combinedErr error

	collectError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		combinedErr = errors.Join(combinedErr, err)
	}

	for _, proc := range lm.processList {
		wg.Add(1)
		go func(p *process.Process) {
			defer wg.Done()
			if !p.IsRunning() {
				return
			}
			if err := p.Stop(); err != nil {
				collectError(err)
			}
		}(proc)
	}

	wg.Wait()
	close(lm.errChan)
	return combinedErr
}

func (lm *LauncherManager) WakeupService(ctx context.Context, serviceID string) (string, int, error) {
	lm.rwMtx.Lock()

	// Si ya está corriendo
	if proc, exists := lm.processList[serviceID]; exists && proc.IsRunning() {
		lm.rwMtx.Unlock()
		service, err := lm.repository.FindService(ctx, serviceID)
		if err != nil {
			return "", 0, err
		}
		port, _ := strconv.Atoi(service.Port)
		if err := lm.waitForHealthCheck(ctx, service.IP, service.Port); err != nil {
			return "", 0, err
		}
		return service.IP, port, nil
	}

	// Cargar el servicio
	service, err := lm.repository.FindService(ctx, serviceID)
	if err != nil {
		lm.rwMtx.Unlock()
		return "", 0, err
	}
	if service.Deleted != "" {
		lm.rwMtx.Unlock()
		return "", 0, fmt.Errorf("service is deleted")
	}
	if service.Status == models.Stopped {
		lm.rwMtx.Unlock()
		return "", 0, fmt.Errorf("service is stopped or paused")
	}

	// Iniciar
	err = lm.startServiceLocked(ctx, *service)
	lm.rwMtx.Unlock()
	if err != nil {
		return "", 0, err
	}

	// Leer IP y puerto actuales
	service, err = lm.repository.FindService(ctx, serviceID)
	if err != nil {
		return "", 0, err
	}
	port, _ := strconv.Atoi(service.Port)
	if service.IP == "" || port == 0 {
		return "", 0, fmt.Errorf("service started but IP/Port not registered")
	}

	// Esperar al healthcheck
	if err := lm.waitForHealthCheck(ctx, service.IP, service.Port); err != nil {
		return "", 0, err
	}

	return service.IP, port, nil
}

func (lm *LauncherManager) waitForHealthCheck(ctx context.Context, ip string, portStr string) error {
	healthURL := fmt.Sprintf("http://%s/api/health", net.JoinHostPort(ip, portStr))
	client := &http.Client{Timeout: 100 * time.Millisecond}

	maxRetries := 75 // 75 * 40ms = 3000ms max
	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
		if err == nil {
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(40 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for health check at %s", healthURL)
}

func (lm *LauncherManager) RecordActivity(serviceID string) {
	lm.rwMtx.Lock()
	defer lm.rwMtx.Unlock()
	lm.activityMap[serviceID] = time.Now()
}

func (lm *LauncherManager) startAutoSleepTicker() {
	ticker := time.NewTicker(lm.checkTickerInterval)
	defer ticker.Stop()

	slog.Info("Auto-Sleep ticker started",
		"checkInterval", lm.checkTickerInterval.String(),
		"idleTimeout", lm.idleTimeout.String(),
	)

	for {
		select {
		case <-lm.stopChan:
			slog.Info("Auto-Sleep ticker stopped")
			return
		case <-ticker.C:
			lm.checkAndSuspendInactiveServices()
		}
	}
}

func (lm *LauncherManager) suspendService(ctx context.Context, serviceID string) error {
	lm.rwMtx.Lock()
	defer lm.rwMtx.Unlock()
	return lm.suspendServiceLocked(ctx, serviceID)
}

func (lm *LauncherManager) suspendServiceLocked(ctx context.Context, serviceID string) error {
	if err := lm.stopProcessOnlyLocked(serviceID); err != nil {
		slog.Error("failed to stop existing process for suspension", "serviceID", serviceID, "error", err)
		return err
	}

	if err := lm.repository.MarkServiceSleeping(ctx, serviceID); err != nil {
		slog.Error("failed to mark service as sleeping", "serviceID", serviceID, "error", err)
	}
	return nil
}

func (lm *LauncherManager) checkAndSuspendInactiveServices() {
	lm.rwMtx.Lock()
	var toSuspend []string
	now := time.Now()

	for id, proc := range lm.processList {
		if !proc.IsRunning() {
			continue
		}
		lastActive, ok := lm.activityMap[id]
		if !ok {
			lm.activityMap[id] = now
			continue
		}

		if now.Sub(lastActive) >= lm.idleTimeout {
			toSuspend = append(toSuspend, id)
		}
	}
	lm.rwMtx.Unlock()

	for _, id := range toSuspend {
		slog.Info("Suspending inactive service", "serviceID", id, "idleTimeout", lm.idleTimeout.String())
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := lm.suspendService(ctx, id); err != nil {
			slog.Error("failed to auto-sleep service", "serviceID", id, "error", err)
		} else {
			slog.Info("Service successfully suspended due to inactivity", "serviceID", id)
		}
		cancel()
	}
}

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
	"runtime"
	"pb_launcher/internal/launcher/domain/services"
	"pb_launcher/internal/operationlog"
	"pb_launcher/utils/iouitls"
	"pb_launcher/utils/networktools"
	"pb_launcher/utils/processstats"
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
	// vacuumingSet registra qué instancias tienen un VACUUM SQLite en curso.
	// WakeupService espera a que finalice antes de arrancar el proceso.
	vacuumingSet sync.Map
	// Callback invocado cuando una instancia se suspende o detiene.
	// Permite que el proxy invalide su cache sin generar una dependencia circular.
	onServiceDeactivated func(serviceID string)
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

// SetOnServiceDeactivated registra un callback que se llama cuando una
// instancia se suspende (auto-sleep) o se detiene manualmente.
// Se usa para que el proxy invalide su cache de ServiceDiscovery.
func (lm *LauncherManager) SetOnServiceDeactivated(fn func(serviceID string)) {
	lm.rwMtx.Lock()
	defer lm.rwMtx.Unlock()
	lm.onServiceDeactivated = fn
}

// notifyDeactivated invoca el callback si está configurado (sin lock; llamar solo con lock ya adquirido o en goroutine propia).
func (lm *LauncherManager) notifyDeactivated(serviceID string) {
	if lm.onServiceDeactivated != nil {
		go lm.onServiceDeactivated(serviceID)
	}
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

	// Configurar límites de recursos dinámicos por instancia con fallback global
	cpuQuota := lm.cpuQuota
	if service.CpuQuota != "" && !strings.EqualFold(service.CpuQuota, "default") {
		cpuQuota = service.CpuQuota
	}
	realCpuQuota := calculatePortabilityCpuQuota(cpuQuota)

	memoryLimit := lm.memoryLimit
	if service.MemoryLimit != "" && !strings.EqualFold(service.MemoryLimit, "default") {
		memoryLimit = service.MemoryLimit
	}
	if strings.EqualFold(memoryLimit, "none") || strings.EqualFold(memoryLimit, "disabled") {
		memoryLimit = ""
	}

	newProcess := process.New(
		service.ID,
		executablePath,
		serveArgs,
		process.WithErrorChan(lm.errChan),
		process.WithStdout(stdout),
		process.WithStderr(lm.lstore.NewWriter(service.ID, logstore.StreamStderr)),
		process.WithCpuQuota(realCpuQuota),
		process.WithMemoryLimit(memoryLimit),
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

	// Registrar en el monitor de CPU (event-driven, igual que htop al detectar un proceso nuevo)
	processstats.DefaultMonitor.Register(newProcess.GetPID())

	if err := lm.repository.MarkServiceRunning(ctx, service.ID, ip, fmt.Sprint(port)); err != nil {
		slog.Error("failed to update service status to running",
			"serviceID", service.ID,
			"ip", ip,
			"port", port,
			"error", err,
		)
	}

	// Aplicar límite de CPU dinámicamente post-healthcheck en segundo plano (arranque al 100% libre)
	go func() {
		bgCtx, bgCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer bgCancel()
		if err := lm.waitForHealthCheck(bgCtx, ip, fmt.Sprint(port)); err == nil {
			_ = newProcess.SetCpuQuota(realCpuQuota)
		}
	}()

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
		return nil
	}
	if !existingProcess.IsRunning() {
		processstats.DefaultMonitor.Unregister(existingProcess.GetPID())
		delete(lm.processList, serviceID)
		delete(lm.activityMap, serviceID)
		return nil
	}

	pid := existingProcess.GetPID()
	if err := existingProcess.Stop(); err != nil {
		return err
	}

	// Desregistrar del monitor de CPU al detener el proceso
	processstats.DefaultMonitor.Unregister(pid)
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

	lm.notifyDeactivated(serviceID)
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
	// Esperar a que finalice cualquier VACUUM en curso para esta instancia
	// (evita SQLITE_BUSY al arrancar PocketBase mientras se compacta el .db).
	const vacuumWaitTimeout = 30 * time.Second
	const vacuumPollInterval = 200 * time.Millisecond
	deadline := time.Now().Add(vacuumWaitTimeout)
	for lm.IsVacuuming(serviceID) {
		if time.Now().After(deadline) {
			slog.Warn("wakeup: timeout esperando que termine el vacuum, continuando de todas formas",
				"serviceID", serviceID)
			break
		}
		time.Sleep(vacuumPollInterval)
	}

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
	wasSleeping := service.Status == models.Sleeping
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

	// Una vez levantada y sana la instancia, aplicamos el límite configurado de CPU quota
	lm.rwMtx.RLock()
	proc, exists := lm.processList[serviceID]
	lm.rwMtx.RUnlock()
	if exists && proc.IsRunning() {
		cpuQuota := lm.cpuQuota
		if service.CpuQuota != "" && !strings.EqualFold(service.CpuQuota, "default") {
			cpuQuota = service.CpuQuota
		}
		realCpuQuota := calculatePortabilityCpuQuota(cpuQuota)
		go func() {
			_ = proc.SetCpuQuota(realCpuQuota)
		}()
	}

	if wasSleeping {
		lm.operationLogger.Success(ctx, serviceID, "wakeup", "service woken up by incoming request", nil)
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

	lm.operationLogger.Success(ctx, serviceID, "sleep", "service suspended due to inactivity (auto-sleep)", nil)

	lm.notifyDeactivated(serviceID)
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

// IsServiceRunning comprueba de forma síncrona si el servicio está activo en memoria y corriendo.
func (lm *LauncherManager) IsServiceRunning(serviceID string) bool {
	lm.rwMtx.RLock()
	defer lm.rwMtx.RUnlock()
	proc, exists := lm.processList[serviceID]
	return exists && proc.IsRunning()
}

// DataDir devuelve el directorio raíz donde residen los datos de cada instancia.
func (lm *LauncherManager) DataDir() string {
	return lm.dataDir
}

// LockVacuum marca la instancia como "vacuum en curso". Llamar antes de abrir el .db.
func (lm *LauncherManager) LockVacuum(serviceID string) {
	lm.vacuumingSet.Store(serviceID, struct{}{})
}

// UnlockVacuum libera la marca de vacuum. Llamar siempre con defer.
func (lm *LauncherManager) UnlockVacuum(serviceID string) {
	lm.vacuumingSet.Delete(serviceID)
}

// IsVacuuming informa si hay un VACUUM SQLite activo para la instancia dada.
func (lm *LauncherManager) IsVacuuming(serviceID string) bool {
	_, ok := lm.vacuumingSet.Load(serviceID)
	return ok
}

// GetActiveInstancesCount devuelve el recuento exacto de procesos de servicios activos en memoria.
func (lm *LauncherManager) GetActiveInstancesCount() int {
	lm.rwMtx.RLock()
	defer lm.rwMtx.RUnlock()
	count := 0
	for _, proc := range lm.processList {
		if proc.IsRunning() {
			count++
		}
	}
	return count
}

// GetRunningInstancesPIDs devuelve un mapa con los ID de servicio y sus PIDs activos.
func (lm *LauncherManager) GetRunningInstancesPIDs() map[string]int {
	lm.rwMtx.RLock()
	defer lm.rwMtx.RUnlock()
	pids := make(map[string]int)
	for id, proc := range lm.processList {
		if proc != nil && proc.IsRunning() {
			pids[id] = proc.GetPID()
		}
	}
	return pids
}

// FindServiceForCli busca y devuelve el modelo de servicio para CLI de forma segura.
func (lm *LauncherManager) FindServiceForCli(ctx context.Context, serviceID string) (*models.Service, error) {
	return lm.repository.FindService(ctx, serviceID)
}

// FindBinaryPath busca o descarga el binario del servicio de forma segura y DRY.
func (lm *LauncherManager) FindBinaryPath(ctx context.Context, service models.Service) (string, error) {
	return lm.findOrDownloadBinary(ctx, service)
}

// BuildServiceArgs expone buildArgs públicamente bajo un nombre descriptivo y DRY.
func (lm *LauncherManager) BuildServiceArgs(serviceID string) ([]string, error) {
	return lm.buildArgs(serviceID)
}

// calculatePortabilityCpuQuota calcula la cuota cgroups de CPU dinámicamente en base a las vCPUs disponibles del host.
func calculatePortabilityCpuQuota(quotaStr string) string {
	quotaStr = strings.TrimSpace(quotaStr)
	if quotaStr == "" || strings.EqualFold(quotaStr, "none") || strings.EqualFold(quotaStr, "disabled") {
		return ""
	}
	if strings.HasSuffix(quotaStr, "%") {
		percentVal, err := strconv.ParseFloat(strings.TrimSuffix(quotaStr, "%"), 64)
		if err == nil && percentVal > 0 {
			// runtime.NumCPU() devuelve el número de CPUs lógicas (vCPUs) en el sistema actual.
			numCpus := runtime.NumCPU()
			calculatedPercent := float64(numCpus) * percentVal
			return fmt.Sprintf("%.0f%%", calculatedPercent)
		}
	}
	return quotaStr
}



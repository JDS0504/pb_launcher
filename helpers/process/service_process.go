package process

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

type ProcessErrorMessage struct {
	ID    string
	Error error
}

type ProcessOptions struct {
	errChan     chan<- ProcessErrorMessage
	stderr      io.Writer
	stdout      io.Writer
	cpuQuota    string
	memoryLimit string
}

type ProcessOption = func(*ProcessOptions)

func WithErrorChan(errChan chan<- ProcessErrorMessage) ProcessOption {
	return func(options *ProcessOptions) { options.errChan = errChan }
}

func WithStdout(w io.Writer) ProcessOption {
	return func(options *ProcessOptions) { options.stdout = w }
}

func WithStderr(w io.Writer) ProcessOption {
	return func(options *ProcessOptions) { options.stderr = w }
}

func WithCpuQuota(quota string) ProcessOption {
	return func(options *ProcessOptions) { options.cpuQuota = quota }
}

func WithMemoryLimit(limit string) ProcessOption {
	return func(options *ProcessOptions) { options.memoryLimit = limit }
}

type Process struct {
	id      string
	options *ProcessOptions

	command string
	args    []string

	h *handler

	closeChan chan struct{}
	isSystemd bool
}

func New(ID string, command string, args []string, options ...ProcessOption) *Process {
	p := &Process{
		id:      ID,
		h:       &handler{status: Stopped},
		command: command,
		args:    args,
		options: &ProcessOptions{},
	}

	for _, applay := range options {
		applay(p.options)
	}

	return p
}

func (p *Process) Status() ProcessState { return p.h.currentState() }
func (p *Process) IsRunning() bool      { return p.Status() == Running }

func (p *Process) GetPID() int {
	cmd := p.h.currentCommand()
	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Pid
	}
	return 0
}

func (p *Process) Start() error {
	currentState := p.Status()
	if currentState != Stopped {
		return nil
	}

	p.closeChan = make(chan struct{})

	command := p.command
	args := p.args

	if runtime.GOOS == "linux" && (p.options.cpuQuota != "" || p.options.memoryLimit != "") {
		if systemdRunPath, err := exec.LookPath("systemd-run"); err == nil {
			unitName := fmt.Sprintf("pblauncher-%s", p.id)
			p.isSystemd = true
			
			// Limpiar de forma segura cualquier unidad scope anterior en memoria o estado failed
			_ = exec.Command("systemctl", "stop", unitName+".scope").Run()
			_ = exec.Command("systemctl", "reset-failed", unitName+".scope").Run()

			systemdArgs := []string{
				"--scope",
				"--unit=" + unitName,
			}
			// NO pasamos el CPUQuota aquí. Iniciamos al 100% de CPU para arranque instantáneo (0ms latency).
			if p.options.memoryLimit != "" {
				systemdArgs = append(systemdArgs, "-p", "MemoryMax="+p.options.memoryLimit)
			}
			systemdArgs = append(systemdArgs, command)
			systemdArgs = append(systemdArgs, args...)
			command = systemdRunPath
			args = systemdArgs
		}
	}

	cmd := exec.Command(command, args...)
	cmd.Env = []string{}
	if p.options.stdout != nil {
		cmd.Stdout = p.options.stdout
	}
	if p.options.stderr != nil {
		cmd.Stderr = p.options.stderr
	}

	p.h.updateStatus(Starting)
	if err := cmd.Start(); err != nil {
		p.h.updateStatus(Stopped)
		slog.Error("failed to start process", "error", err, "process_id", p.id)
		return err
	}

	go p.waitForExit(cmd, p.closeChan)

	p.h.replaceCommand(cmd)
	p.h.updateStatus(Running)
	return nil
}

func (p *Process) waitForExit(cmd *exec.Cmd, doneChan chan struct{}) {
	err := cmd.Wait()
	currentState := p.Status()

	if err != nil && currentState != Stopping && currentState != Stopped {
		if err.Error() != "signal: terminated" {
			if p.options.errChan != nil {
				p.options.errChan <- ProcessErrorMessage{
					ID:    p.id,
					Error: fmt.Errorf("process exited with error: %w", err),
				}
			}
			slog.Error("process exited with error", "error", err, "process_id", p.id)
		}
	}
	p.h.updateStatus(Stopped)
	if doneChan != nil {
		close(doneChan)
	}
}

func (p *Process) Stop() error {
	currentState := p.Status()
	if currentState != Running {
		return nil
	}

	cmd := p.h.currentCommand()
	if cmd == nil {
		slog.Warn("stop ignored: no active command found", "process_id", p.id)
		return nil
	}

	p.h.updateStatus(Stopping)

	var stopErr error
	if p.isSystemd {
		unitName := fmt.Sprintf("pblauncher-%s", p.id)
		slog.Info("stopping systemd scope unit", "unit", unitName, "process_id", p.id)
		stopErr = exec.Command("systemctl", "stop", unitName+".scope").Run()
		_ = exec.Command("systemctl", "reset-failed", unitName+".scope").Run()
		if stopErr != nil {
			slog.Warn("systemctl stop failed, falling back to SIGTERM", "error", stopErr, "process_id", p.id)
			stopErr = cmd.Process.Signal(syscall.SIGTERM)
		}
	} else if runtime.GOOS == "windows" {
		stopErr = cmd.Process.Kill()
	} else {
		stopErr = cmd.Process.Signal(syscall.SIGTERM)
	}

	if stopErr != nil {
		p.h.updateStatus(Running)
		slog.Error("failed to stop process", "error", stopErr, "process_id", p.id)
		return stopErr
	}

	if p.closeChan != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		select {
		case <-p.closeChan:
		case <-ctx.Done():
			slog.Warn("process did not exit gracefully, sending SIGKILL", "process_id", p.id)
			_ = cmd.Process.Kill()
		}
	}
	return nil
}

// SetCpuQuota aplica dinámicamente el límite de CPU en caliente al Scope de Systemd (KISS).
// Permite que el proceso inicie al 100% y sea limitado tras el healthcheck exitoso.
func (p *Process) SetCpuQuota(quota string) error {
	if !p.isSystemd || quota == "" {
		return nil
	}
	unitName := fmt.Sprintf("pblauncher-%s", p.id)
	slog.Info("applying dynamic CPU limit", "unit", unitName, "quota", quota)
	cmd := exec.Command("systemctl", "set-property", "--runtime", unitName+".scope", "CPUQuota="+quota)
	if err := cmd.Run(); err != nil {
		slog.Error("failed to apply dynamic CPU quota", "unit", unitName, "error", err)
		return err
	}
	return nil
}


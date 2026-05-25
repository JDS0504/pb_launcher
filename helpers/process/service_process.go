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
			
			// Limpiar de forma segura cualquier unidad scope anterior en memoria o estado failed
			_ = exec.Command("systemctl", "stop", unitName+".scope").Run()
			_ = exec.Command("systemctl", "reset-failed", unitName+".scope").Run()

			systemdArgs := []string{
				"--scope",
				"--unit=" + unitName,
			}
			if p.options.cpuQuota != "" {
				systemdArgs = append(systemdArgs, "-p", "CPUQuota="+p.options.cpuQuota)
			}
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
	if err := cmd.Wait(); err != nil {
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
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		p.h.updateStatus(Running)
		slog.Error("failed to stop process", "error", err, "process_id", p.id)
		return err
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

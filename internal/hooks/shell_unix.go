//go:build !windows

package hooks

import (
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

func handleShellSession(conn *websocket.Conn) {
	shell := detectShell()

	cmd := exec.Command(shell)
	cmd.Env = buildShellEnv()

	// Iniciar el comando con un pseudo-terminal UNIX real
	ptyF, err := pty.Start(cmd)
	if err != nil {
		writeWSText(conn, "\r\n\033[31m[ERROR] No se pudo iniciar la PTY: "+err.Error()+"\033[0m\r\n")
		return
	}
	defer ptyF.Close()

	slog.Info("shell (unix): process started with PTY", "shell", shell, "pid", cmd.Process.Pid)

	// Banner de bienvenida
	writeWSText(conn, "\033[1;32m╔══════════════════════════════════════════╗\033[0m\r\n")
	writeWSText(conn, "\033[1;32m║     PB Launcher · Shell Interactiva      ║\033[0m\r\n")
	writeWSText(conn, "\033[1;32m║  Solo accesible para administradores     ║\033[0m\r\n")
	writeWSText(conn, "\033[1;32m║  PTY Real Activa · Timeout: 30 minutos   ║\033[0m\r\n")
	writeWSText(conn, "\033[1;32m╚══════════════════════════════════════════╝\033[0m\r\n\r\n")

	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() { closeOnce.Do(func() { close(done) }) }

	// PTY stdout/stderr -> WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptyF.Read(buf)
			if n > 0 {
				conn.SetWriteDeadline(time.Now().Add(shellWriteTimeout))
				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		closeDone()
	}()

	// Esperar fin del proceso
	go func() {
		_ = cmd.Wait()
		writeWSText(conn, "\r\n\033[33m[SHELL] Sesión terminada.\033[0m\r\n")
		time.Sleep(100 * time.Millisecond)
		closeDone()
	}()

	sessionTimer := time.NewTimer(shellSessionTimeout)
	defer sessionTimer.Stop()

	// WebSocket -> PTY stdin
	conn.SetReadDeadline(time.Now().Add(shellSessionTimeout))
	for {
		select {
		case <-done:
			return
		case <-sessionTimer.C:
			writeWSText(conn, "\r\n\033[33m[TIMEOUT] Sesión expirada por inactividad (30 min).\033[0m\r\n")
			_ = cmd.Process.Kill()
			return
		default:
		}

		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Reiniciar temporizador de inactividad
		conn.SetReadDeadline(time.Now().Add(shellSessionTimeout))
		if !sessionTimer.Stop() {
			select {
			case <-sessionTimer.C:
			default:
			}
		}
		sessionTimer.Reset(shellSessionTimeout)

		if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
			_, _ = ptyF.Write(msg)
		}
	}

	_ = cmd.Process.Kill()
}

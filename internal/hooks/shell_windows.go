//go:build windows

package hooks

import (
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func handleShellSession(conn *websocket.Conn) {
	// Fallback para Windows usando cmd.exe y pipes estándar (ya que no soporta UNIX PTY)
	shell := "cmd.exe"
	cmd := exec.Command(shell)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		writeWSText(conn, "\r\n\033[31m[ERROR] No se pudo abrir stdin: "+err.Error()+"\033[0m\r\n")
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeWSText(conn, "\r\n\033[31m[ERROR] No se pudo abrir stdout: "+err.Error()+"\033[0m\r\n")
		stdin.Close()
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		writeWSText(conn, "\r\n\033[31m[ERROR] No se pudo abrir stderr: "+err.Error()+"\033[0m\r\n")
		stdin.Close()
		return
	}

	if err := cmd.Start(); err != nil {
		writeWSText(conn, "\r\n\033[31m[ERROR] No se pudo iniciar cmd.exe: "+err.Error()+"\033[0m\r\n")
		return
	}

	slog.Info("shell (windows): process started", "pid", cmd.Process.Pid)

	// Banner de bienvenida fallback
	writeWSText(conn, "\033[1;32m╔══════════════════════════════════════════╗\033[0m\r\n")
	writeWSText(conn, "\033[1;32m║     PB Launcher · Shell Windows (cmd)    ║\033[0m\r\n")
	writeWSText(conn, "\033[1;32m║  Pipes fallback · Timeout: 30 minutos   ║\033[0m\r\n")
	writeWSText(conn, "\033[1;32m╚══════════════════════════════════════════╝\033[0m\r\n\r\n")

	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() { closeOnce.Do(func() { close(done) }) }

	var wg sync.WaitGroup

	// stdout -> WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := stdout.Read(buf)
			if n > 0 {
				conn.SetWriteDeadline(time.Now().Add(shellWriteTimeout))
				if wsErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); wsErr != nil {
					break
				}
			}
			if readErr != nil {
				break
			}
		}
	}()

	// stderr -> WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := stderr.Read(buf)
			if n > 0 {
				conn.SetWriteDeadline(time.Now().Add(shellWriteTimeout))
				_ = conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			}
			if readErr != nil {
				break
			}
		}
	}()

	// Esperar fin del proceso
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = cmd.Wait()
		writeWSText(conn, "\r\n\033[33m[SHELL] Sesión terminada.\033[0m\r\n")
		time.Sleep(100 * time.Millisecond)
		closeDone()
	}()

	sessionTimer := time.NewTimer(shellSessionTimeout)
	defer sessionTimer.Stop()

	// WebSocket -> stdin
	conn.SetReadDeadline(time.Now().Add(shellSessionTimeout))
	for {
		select {
		case <-done:
			stdin.Close()
			wg.Wait()
			return
		case <-sessionTimer.C:
			writeWSText(conn, "\r\n\033[33m[TIMEOUT] Sesión expirada.\033[0m\r\n")
			_ = cmd.Process.Kill()
			stdin.Close()
			wg.Wait()
			return
		default:
		}

		msgType, msg, readErr := conn.ReadMessage()
		if readErr != nil {
			break
		}

		conn.SetReadDeadline(time.Now().Add(shellSessionTimeout))
		if !sessionTimer.Stop() {
			select {
			case <-sessionTimer.C:
			default:
			}
		}
		sessionTimer.Reset(shellSessionTimeout)

		if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
			_, _ = stdin.Write(msg)
		}
	}

	_ = cmd.Process.Kill()
	stdin.Close()
	closeDone()
	wg.Wait()
}

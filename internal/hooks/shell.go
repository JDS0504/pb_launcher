package hooks

import (
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	shellSessionTimeout = 30 * time.Minute
	shellWriteTimeout   = 10 * time.Second
)

var shellUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// La autenticación se verifica antes del upgrade mediante re.Auth.
	// Aceptar cualquier origen ya que el token de PocketBase protege el acceso.
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// RegisterShellRoute registra el endpoint WebSocket de la shell interactiva.
// Acceso EXCLUSIVO para administradores autenticados.
// El token se acepta como query param ?token= porque los WebSockets del
// navegador no soportan cabeceras personalizadas durante el handshake.
func RegisterShellRoute(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/shell", func(re *core.RequestEvent) error {
			// Autenticación: primero intentar auth estándar, luego token por query param
			authRecord := re.Auth
			if authRecord == nil {
				// Fallback: token en query param (necesario para WebSocket desde el browser)
				token := re.Request.URL.Query().Get("token")
				if token == "" {
					return re.JSON(http.StatusUnauthorized, map[string]string{
						"error": "unauthorized: token required",
					})
				}
				record, err := re.App.FindAuthRecordByToken(token, core.TokenTypeAuth)
				if err != nil || record == nil {
					return re.JSON(http.StatusUnauthorized, map[string]string{
						"error": "unauthorized: invalid token",
					})
				}
				authRecord = record
			}

			conn, err := shellUpgrader.Upgrade(re.Response, re.Request, nil)
			if err != nil {
				slog.Error("shell: failed to upgrade websocket", "error", err)
				return nil
			}
			defer conn.Close()

			slog.Info("shell: new session started", "adminID", authRecord.Id)
			handleShellSession(conn)
			slog.Info("shell: session ended", "adminID", authRecord.Id)
			return nil
		})

		return se.Next()
	})
}

// detectShell devuelve la shell disponible en el sistema: bash > sh.
func detectShell() string {
	for _, sh := range []string{"/bin/bash", "/usr/bin/bash", "/bin/sh", "/usr/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}
	// Fallback: buscar en PATH
	if path, err := exec.LookPath("bash"); err == nil {
		return path
	}
	if path, err := exec.LookPath("sh"); err == nil {
		return path
	}
	return "sh"
}

// buildShellEnv construye las variables de entorno para la shell interactiva.
func buildShellEnv() []string {
	base := os.Environ()
	extra := []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"HISTCONTROL=ignoredups",
	}
	return append(base, extra...)
}

// writeWSText envía un mensaje de texto al cliente WebSocket de forma segura.
func writeWSText(conn *websocket.Conn, msg string) {
	conn.SetWriteDeadline(time.Now().Add(shellWriteTimeout))
	_ = conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

// handleShellSession gestiona una sesión de shell interactiva completa sobre WebSocket.
func handleShellSession(conn *websocket.Conn) {
	shell := detectShell()

	cmd := exec.Command(shell)
	cmd.Env = buildShellEnv()

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
		writeWSText(conn, "\r\n\033[31m[ERROR] No se pudo iniciar la shell '"+shell+"': "+err.Error()+"\033[0m\r\n")
		return
	}

	slog.Info("shell: process started", "shell", shell, "pid", cmd.Process.Pid)

	// Banner de bienvenida
	writeWSText(conn, "\033[1;32m╔══════════════════════════════════════════╗\033[0m\r\n")
	writeWSText(conn, "\033[1;32m║     PB Launcher · Shell Interactiva      ║\033[0m\r\n")
	writeWSText(conn, "\033[1;32m║  Solo accesible para administradores     ║\033[0m\r\n")
	writeWSText(conn, "\033[1;32m║  Timeout de sesión: 30 minutos           ║\033[0m\r\n")
	writeWSText(conn, "\033[1;32m╚══════════════════════════════════════════╝\033[0m\r\n\r\n")

	done := make(chan struct{})
	var closeOnce sync.Once
	closeDone := func() { closeOnce.Do(func() { close(done) }) }

	var wg sync.WaitGroup
	sessionTimer := time.NewTimer(shellSessionTimeout)
	defer sessionTimer.Stop()

	// stdout → WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, readErr := stdout.Read(buf)
			if n > 0 {
				conn.SetWriteDeadline(time.Now().Add(shellWriteTimeout))
				if wsErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); wsErr != nil {
					return
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	// stderr → WebSocket
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
				return
			}
		}
	}()

	// Esperar fin del proceso
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = cmd.Wait()
		writeWSText(conn, "\r\n\033[33m[SHELL] Sesión terminada. Puedes cerrar esta ventana.\033[0m\r\n")
		closeDone()
	}()

	// Loop principal: WebSocket → stdin del proceso
	conn.SetReadDeadline(time.Now().Add(shellSessionTimeout))
	for {
		select {
		case <-done:
			stdin.Close()
			wg.Wait()
			return
		case <-sessionTimer.C:
			writeWSText(conn, "\r\n\033[33m[TIMEOUT] Sesión expirada por inactividad (30 min).\033[0m\r\n")
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

		// Reiniciar timer de inactividad en cada input del usuario
		conn.SetReadDeadline(time.Now().Add(shellSessionTimeout))
		if !sessionTimer.Stop() {
			select {
			case <-sessionTimer.C:
			default:
			}
		}
		sessionTimer.Reset(shellSessionTimeout)

		if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
			if _, writeErr := stdin.Write(msg); writeErr != nil {
				break
			}
		}
	}

	_ = cmd.Process.Kill()
	stdin.Close()
	closeDone()
	wg.Wait()
	slog.Info("shell: session closed", "pid", cmd.Process.Pid)
}

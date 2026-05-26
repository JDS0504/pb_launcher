package hooks

import (
	"log/slog"
	"net/http"
	"os"
	"os/exec"
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
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// RegisterShellRoute registra el endpoint WebSocket de la shell interactiva.
func RegisterShellRoute(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/shell", func(re *core.RequestEvent) error {
			authRecord := re.Auth
			if authRecord == nil {
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

func detectShell() string {
	for _, sh := range []string{"/bin/bash", "/usr/bin/bash", "/bin/sh", "/usr/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}
	if path, err := exec.LookPath("bash"); err == nil {
		return path
	}
	if path, err := exec.LookPath("sh"); err == nil {
		return path
	}
	return "sh"
}

func buildShellEnv() []string {
	base := os.Environ()
	extra := []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"HISTCONTROL=ignoredups",
	}
	return append(base, extra...)
}

func writeWSText(conn *websocket.Conn, msg string) {
	conn.SetWriteDeadline(time.Now().Add(shellWriteTimeout))
	_ = conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

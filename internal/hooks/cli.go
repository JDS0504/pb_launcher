package hooks

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	launcher "pb_launcher/internal/launcher/domain"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterServiceCliRoute(app *pocketbase.PocketBase, launcherManager *launcher.LauncherManager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/x-api/service/cli/{service_id}", func(re *core.RequestEvent) error {
			// Seguridad: Verificar que el usuario sea administrador autenticado
			if re.Auth == nil {
				return re.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized: admin access required"})
			}

			serviceID := re.Request.PathValue("service_id")
			if serviceID == "" {
				return re.BadRequestError("service_id required", nil)
			}

			var body struct {
				Args []string `json:"args"`
			}
			if err := re.BindBody(&body); err != nil {
				return re.BadRequestError("invalid JSON body", err)
			}

			if len(body.Args) == 0 {
				return re.BadRequestError("args list cannot be empty", nil)
			}

			// Validaciones preventivas para evitar inyección y comandos peligrosos a nivel de binario
			// No permitimos lanzar el comando "serve" directamente en segundo plano de esta forma interactiva
			forbiddenCmds := map[string]bool{
				"serve": true,
			}
			if forbiddenCmds[strings.ToLower(body.Args[0])] {
				return re.BadRequestError(fmt.Sprintf("command '%s' is not allowed in interactive mode", body.Args[0]), nil)
			}

			// Buscar servicio e instanciar
			ctx, cancel := context.WithTimeout(re.Request.Context(), 45*time.Second)
			defer cancel()

			// Obtener binario de la instancia mediante el launcher manager
			svcModel, err := launcherManager.FindServiceForCli(ctx, serviceID)
			if err != nil {
				return re.InternalServerError("failed to load service models", err)
			}

			binaryPath, err := launcherManager.FindBinaryPath(ctx, *svcModel)
			if err != nil {
				return re.InternalServerError("failed to find PocketBase executable path", err)
			}

			// Construir argumentos base (--dir, --hooksDir, --publicDir, --migrationsDir)
			baseArgs, err := launcherManager.BuildServiceArgs(svcModel.Name)
			if err != nil {
				return re.InternalServerError("failed to build default args", err)
			}

			// Unir argumentos base y el comando enviado por el usuario
			finalArgs := append(baseArgs, body.Args...)

			// Ejecutar el binario de forma aislada y controlada con timeout
			cmd := exec.CommandContext(ctx, binaryPath, finalArgs...)
			output, execErr := cmd.CombinedOutput()

			resultStr := string(output)
			if execErr != nil {
				if resultStr == "" {
					resultStr = execErr.Error()
				}
				return re.JSON(http.StatusOK, map[string]any{
					"success": false,
					"output":  resultStr,
				})
			}

			return re.JSON(http.StatusOK, map[string]any{
				"success": true,
				"output":  resultStr,
			})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

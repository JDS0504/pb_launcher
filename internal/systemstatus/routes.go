package systemstatus

import (
	"net/http"
	launcherdomain "pb_launcher/internal/launcher/domain"
	"pb_launcher/utils/processstats"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes registers the /x-api/system/status route in PocketBase.
// Usa el DefaultMonitor de processstats para obtener las métricas de CPU sin bloquear la API.
func RegisterRoutes(app *pocketbase.PocketBase, launcherManager *launcherdomain.LauncherManager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/system/status", func(e *core.RequestEvent) error {
			// Obtener PIDs activos y sincronizar el monitor
			activePIDs := launcherManager.GetRunningInstancesPIDs()
			processstats.DefaultMonitor.SyncPIDs(activePIDs)

			// Recopilar métricas del sistema (disco, RAM, host)
			activeInstances := launcherManager.GetActiveInstancesCount()
			status, err := CollectStatus(".")
			if err != nil {
				return e.BadRequestError("failed to collect system status", err)
			}
			status.Host.ActiveInstances = activeInstances

			// Devolver la última lectura de CPU de cada instancia (0ms de latencia)
			status.InstancesStats = make(map[string]InstanceStats)
			for id, pid := range activePIDs {
				status.InstancesStats[id] = processstats.DefaultMonitor.Get(pid)
			}

			return e.JSON(http.StatusOK, status)
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

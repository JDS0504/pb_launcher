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
func RegisterRoutes(app *pocketbase.PocketBase, launcherManager *launcherdomain.LauncherManager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/system/status", func(e *core.RequestEvent) error {
			activeInstances := launcherManager.GetActiveInstancesCount()
			status, err := CollectStatus(".")
			if err != nil {
				return e.BadRequestError("failed to collect system status", err)
			}
			status.Host.ActiveInstances = activeInstances

			// Leer métricas de CPU desde el monitor (0ms de latencia, sin bloqueos)
			activePIDs := launcherManager.GetRunningInstancesPIDs()
			status.InstancesStats = make(map[string]InstanceStats, len(activePIDs))
			for id, pid := range activePIDs {
				status.InstancesStats[id] = processstats.DefaultMonitor.Get(pid)
			}

			return e.JSON(http.StatusOK, status)
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

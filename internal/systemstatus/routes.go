package systemstatus

import (
	"net/http"
	launcherdomain "pb_launcher/internal/launcher/domain"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes registers the /x-api/system/status route in PocketBase
func RegisterRoutes(app *pocketbase.PocketBase, launcherManager *launcherdomain.LauncherManager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/system/status", func(e *core.RequestEvent) error {
			// Get number of active instances running currently in memory
			activeInstances := launcherManager.GetActiveInstancesCount()

			// Collect system metrics with "." representing the primary storage partition
			status, err := CollectStatus(".")
			if err != nil {
				return e.BadRequestError("failed to collect system status", err)
			}

			// Populate dynamic metadata
			status.Host.ActiveInstances = activeInstances

			return e.JSON(http.StatusOK, status)
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

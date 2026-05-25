package systemstatus

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// RegisterRoutes registers the /x-api/system/status route in PocketBase
func RegisterRoutes(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/system/status", func(e *core.RequestEvent) error {
			// Query number of active instances running currently
			var activeInstances int
			err := app.DB().
				Select("count(*)").
				From("services").
				Where(dbx.HashExp{"status": "running", "deleted": ""}).
				Row(&activeInstances)
			if err != nil {
				activeInstances = 0
			}

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

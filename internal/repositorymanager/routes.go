package repositorymanager

import (
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterRoutes(app *pocketbase.PocketBase, manager *Manager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/repositories/status", func(e *core.RequestEvent) error {
			statuses, err := manager.ListStatus(e.Request.Context())
			if err != nil {
				return e.InternalServerError("failed to list repository sync status", err)
			}
			return e.JSON(http.StatusOK, statuses)
		}).Bind(apis.RequireAuth())

		se.Router.POST("/x-api/repositories/{repository_id}/sync", func(e *core.RequestEvent) error {
			if err := manager.Sync(e.Request.Context(), e.Request.PathValue("repository_id")); err != nil {
				return e.BadRequestError("failed to sync repository", err)
			}
			return e.NoContent(http.StatusOK)
		}).Bind(apis.RequireAuth())
		return se.Next()
	})
}

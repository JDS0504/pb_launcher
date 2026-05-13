package hookmanager

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterRoutes(app *pocketbase.PocketBase, manager *Manager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/services/{service_id}/hooks", func(e *core.RequestEvent) error {
			files, err := manager.List(e.Request.Context(), e.Request.PathValue("service_id"))
			if err != nil {
				return e.BadRequestError("failed to list PB hooks", err)
			}
			return e.JSON(http.StatusOK, files)
		}).Bind(apis.RequireAuth())

		se.Router.GET("/x-api/services/{service_id}/hooks/export", func(e *core.RequestEvent) error {
			exportFile, err := manager.Export(e.Request.Context(), e.Request.PathValue("service_id"))
			if err != nil {
				return e.BadRequestError("failed to export PB hooks", err)
			}
			defer os.Remove(exportFile.Path)

			e.Response.Header().Set("Content-Type", "application/zip")
			e.Response.Header().Set("Content-Disposition", `attachment; filename="`+exportFile.Filename+`"`)
			http.ServeFile(e.Response, e.Request, exportFile.Path)
			return nil
		}).Bind(apis.RequireAuth())

		se.Router.GET("/x-api/services/{service_id}/hooks/file", func(e *core.RequestEvent) error {
			file, err := manager.ReadFile(
				e.Request.Context(),
				e.Request.PathValue("service_id"),
				strings.TrimSpace(e.Request.URL.Query().Get("path")),
			)
			if err != nil {
				return e.BadRequestError("failed to read PB hook", err)
			}
			return e.JSON(http.StatusOK, file)
		}).Bind(apis.RequireAuth())

		se.Router.PUT("/x-api/services/{service_id}/hooks/file", func(e *core.RequestEvent) error {
			var body struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := e.BindBody(&body); err != nil {
				return e.BadRequestError("invalid JSON body", err)
			}
			if err := manager.SaveFile(e.Request.Context(), e.Request.PathValue("service_id"), body.Path, body.Content); err != nil {
				return e.BadRequestError("failed to save PB hook", err)
			}
			return e.NoContent(http.StatusOK)
		}).Bind(apis.RequireAuth())

		se.Router.DELETE("/x-api/services/{service_id}/hooks/file", func(e *core.RequestEvent) error {
			if err := manager.DeleteFile(
				e.Request.Context(),
				e.Request.PathValue("service_id"),
				strings.TrimSpace(e.Request.URL.Query().Get("path")),
			); err != nil {
				return e.BadRequestError("failed to delete PB hook", err)
			}
			return e.NoContent(http.StatusOK)
		}).Bind(apis.RequireAuth())

		se.Router.POST("/x-api/services/{service_id}/hooks/import", func(e *core.RequestEvent) error {
			file, _, err := e.Request.FormFile("hooks")
			if err != nil {
				return e.BadRequestError("hooks zip file is required", err)
			}
			defer file.Close()

			tempFile, err := os.CreateTemp("", "pblauncher-hooks-upload-*.zip")
			if err != nil {
				return e.InternalServerError("failed to create temporary upload file", err)
			}
			defer os.Remove(tempFile.Name())

			if _, err := io.Copy(tempFile, file); err != nil {
				tempFile.Close()
				return e.InternalServerError("failed to store uploaded hooks", err)
			}
			if err := tempFile.Close(); err != nil {
				return e.InternalServerError("failed to close uploaded hooks", err)
			}

			files, err := manager.Import(e.Request.Context(), e.Request.PathValue("service_id"), tempFile.Name())
			if err != nil {
				return e.BadRequestError("failed to import PB hooks", err)
			}
			return e.JSON(http.StatusOK, map[string]any{"files": files, "count": len(files)})
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

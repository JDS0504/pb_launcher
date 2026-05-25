package filemanager

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func RegisterRoutes(app *pocketbase.PocketBase, manager *Manager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/x-api/services/{service_id}/files", func(e *core.RequestEvent) error {
			files, err := manager.List(e.Request.Context(), e.Request.PathValue("service_id"))
			if err != nil {
				return e.BadRequestError("failed to list files", err)
			}
			return e.JSON(http.StatusOK, files)
		}).Bind(apis.RequireAuth())

		se.Router.GET("/x-api/services/{service_id}/files/content", func(e *core.RequestEvent) error {
			file, err := manager.ReadFile(
				e.Request.Context(),
				e.Request.PathValue("service_id"),
				strings.TrimSpace(e.Request.URL.Query().Get("path")),
			)
			if err != nil {
				return e.BadRequestError("failed to read file", err)
			}
			return e.JSON(http.StatusOK, file)
		}).Bind(apis.RequireAuth())

		se.Router.PUT("/x-api/services/{service_id}/files/content", func(e *core.RequestEvent) error {
			var body struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := e.BindBody(&body); err != nil {
				return e.BadRequestError("invalid JSON body", err)
			}
			if err := manager.SaveFile(e.Request.Context(), e.Request.PathValue("service_id"), body.Path, body.Content); err != nil {
				return e.BadRequestError("failed to save file", err)
			}
			return e.NoContent(http.StatusOK)
		}).Bind(apis.RequireAuth())

		se.Router.POST("/x-api/services/{service_id}/files/upload", func(e *core.RequestEvent) error {
			if err := e.Request.ParseMultipartForm(50 << 20); err != nil {
				return e.BadRequestError("failed to parse multipart form", err)
			}

			destPath := strings.TrimSpace(e.Request.FormValue("path"))
			if destPath == "" {
				destPath = "pb_public"
			}

			multipartForm := e.Request.MultipartForm
			if multipartForm == nil {
				return e.BadRequestError("multipart form is empty", nil)
			}

			files := multipartForm.File["files"]
			if len(files) == 0 {
				return e.BadRequestError("no files provided", nil)
			}

			for _, fh := range files {
				file, err := fh.Open()
				if err != nil {
					return e.BadRequestError(fmt.Sprintf("failed to open file %s", fh.Filename), err)
				}
				defer file.Close()

				data, err := io.ReadAll(file)
				if err != nil {
					return e.BadRequestError(fmt.Sprintf("failed to read file content for %s", fh.Filename), err)
				}

				targetFilePath := filepath.ToSlash(filepath.Join(destPath, fh.Filename))

				if err := manager.SaveFileBytes(e.Request.Context(), e.Request.PathValue("service_id"), targetFilePath, data); err != nil {
					return e.BadRequestError(fmt.Sprintf("failed to save file %s: %v", fh.Filename, err), err)
				}
			}

			return e.NoContent(http.StatusOK)
		}).Bind(apis.RequireAuth())

		se.Router.DELETE("/x-api/services/{service_id}/files", func(e *core.RequestEvent) error {
			if err := manager.DeleteFile(
				e.Request.Context(),
				e.Request.PathValue("service_id"),
				strings.TrimSpace(e.Request.URL.Query().Get("path")),
			); err != nil {
				return e.BadRequestError("failed to delete file", err)
			}
			return e.NoContent(http.StatusOK)
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

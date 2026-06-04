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
			// MultipartReader hace streaming parte a parte sin cargar todo en RAM,
			// a diferencia de ParseMultipartForm que espera recibir el body completo antes de continuar.
			mr, err := e.Request.MultipartReader()
			if err != nil {
				return e.BadRequestError("failed to parse multipart form", err)
			}

			serviceID := e.Request.PathValue("service_id")
			destPath := "pb_public"
			filesUploaded := 0

			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					return e.BadRequestError("failed to read multipart part", err)
				}

				switch part.FormName() {
				case "path":
					// El campo "path" siempre viene antes que los archivos (ver frontend)
					raw, err := io.ReadAll(part)
					if err != nil {
						return e.BadRequestError("failed to read path field", err)
					}
					if v := strings.TrimSpace(string(raw)); v != "" {
						destPath = v
					}

				case "files":
					fileName := part.FileName()
					if fileName == "" {
						continue
					}
					targetFilePath := filepath.ToSlash(filepath.Join(destPath, fileName))
					// Escribe directamente desde la red a disco sin pasar por RAM
					if err := manager.SaveFileStream(e.Request.Context(), serviceID, targetFilePath, part); err != nil {
						return e.BadRequestError(fmt.Sprintf("failed to save file %s: %v", fileName, err), err)
					}
					filesUploaded++
				}
			}

			if filesUploaded == 0 {
				return e.BadRequestError("no files provided", nil)
			}

			return e.NoContent(http.StatusOK)
		}).Bind(apis.RequireAuth(), apis.BodyLimit(300<<20))

		se.Router.GET("/x-api/services/{service_id}/files/download", func(e *core.RequestEvent) error {
			filePath := strings.TrimSpace(e.Request.URL.Query().Get("path"))
			fullPath, err := manager.GetSafeFilePath(e.Request.Context(), e.Request.PathValue("service_id"), filePath)
			if err != nil {
				return e.BadRequestError("failed to download file", err)
			}
			fileName := filepath.Base(fullPath)
			e.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
			http.ServeFile(e.Response, e.Request, fullPath)
			return nil
		}).Bind(apis.RequireAuth())

		se.Router.POST("/x-api/services/{service_id}/files/folder", func(e *core.RequestEvent) error {
			var body struct {
				Path string `json:"path"`
			}
			if err := e.BindBody(&body); err != nil {
				return e.BadRequestError("invalid JSON body", err)
			}
			if err := manager.CreateDirectory(e.Request.Context(), e.Request.PathValue("service_id"), body.Path); err != nil {
				return e.BadRequestError("failed to create directory", err)
			}
			return e.NoContent(http.StatusOK)
		}).Bind(apis.RequireAuth())

		se.Router.PATCH("/x-api/services/{service_id}/files/rename", func(e *core.RequestEvent) error {
			var body struct {
				OldPath string `json:"old_path"`
				NewPath string `json:"new_path"`
			}
			if err := e.BindBody(&body); err != nil {
				return e.BadRequestError("invalid JSON body", err)
			}
			if err := manager.RenameFile(e.Request.Context(), e.Request.PathValue("service_id"), body.OldPath, body.NewPath); err != nil {
				return e.BadRequestError("failed to rename file", err)
			}
			return e.NoContent(http.StatusOK)
		}).Bind(apis.RequireAuth())

		se.Router.POST("/x-api/services/{service_id}/files/unzip", func(e *core.RequestEvent) error {
			var body struct {
				Path string `json:"path"`
			}
			if err := e.BindBody(&body); err != nil {
				return e.BadRequestError("invalid JSON body", err)
			}
			if err := manager.ExtractZip(e.Request.Context(), e.Request.PathValue("service_id"), body.Path); err != nil {
				return e.BadRequestError("failed to extract zip", err)
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

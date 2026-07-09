package backup

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

		// POST /x-api/services/restore — restaura un ZIP externo como nueva instancia
		se.Router.POST("/x-api/services/restore", func(e *core.RequestEvent) error {
			name := strings.TrimSpace(e.Request.FormValue("name"))
			if name == "" {
				return e.BadRequestError("instance name is required", nil)
			}

			file, _, err := e.Request.FormFile("backup")
			if err != nil {
				return e.BadRequestError("backup file is required", err)
			}
			defer file.Close()

			tempFile, err := os.CreateTemp("", "pblauncher-upload-*.zip")
			if err != nil {
				return e.InternalServerError("failed to create temporary upload file", err)
			}
			defer os.Remove(tempFile.Name())

			if _, err := io.Copy(tempFile, file); err != nil {
				tempFile.Close()
				return e.InternalServerError("failed to store uploaded backup", err)
			}
			if err := tempFile.Close(); err != nil {
				return e.InternalServerError("failed to close uploaded backup", err)
			}

			serviceID, err := manager.Restore(e.Request.Context(), tempFile.Name(), name)
			if err != nil {
				return e.BadRequestError("failed to restore backup", err)
			}
			return e.JSON(http.StatusOK, map[string]string{"service_id": serviceID})
		}).Bind(apis.RequireAuth())

		// POST /x-api/services/{service_id}/clone — clona una instancia existente
		se.Router.POST("/x-api/services/{service_id}/clone", func(e *core.RequestEvent) error {
			name := strings.TrimSpace(e.Request.FormValue("name"))
			if name == "" {
				return e.BadRequestError("instance name is required", nil)
			}

			serviceID, err := manager.Clone(e.Request.Context(), e.Request.PathValue("service_id"), name)
			if err != nil {
				return e.BadRequestError("failed to clone service", err)
			}
			return e.JSON(http.StatusOK, map[string]string{"service_id": serviceID})
		}).Bind(apis.RequireAuth())

		// GET /x-api/services/{service_id}/snapshots — lista snapshots del servicio
		se.Router.GET("/x-api/services/{service_id}/snapshots", func(e *core.RequestEvent) error {
			snapshots, err := manager.ListSnapshots(e.Request.Context(), e.Request.PathValue("service_id"))
			if err != nil {
				return e.BadRequestError("failed to list snapshots", err)
			}
			return e.JSON(http.StatusOK, snapshots)
		}).Bind(apis.RequireAuth())

		// POST /x-api/services/{service_id}/snapshots — crea un snapshot manual
		se.Router.POST("/x-api/services/{service_id}/snapshots", func(e *core.RequestEvent) error {
			name := strings.TrimSpace(e.Request.FormValue("name"))
			if name == "" {
				return e.BadRequestError("snapshot name is required", nil)
			}
			comment := strings.TrimSpace(e.Request.FormValue("comment"))

			snapshot, err := manager.CreateSnapshot(e.Request.Context(), e.Request.PathValue("service_id"), name, comment)
			if err != nil {
				return e.BadRequestError("failed to create snapshot", err)
			}
			return e.JSON(http.StatusOK, snapshot)
		}).Bind(apis.RequireAuth())

		// POST /x-api/services/{service_id}/snapshots/{snapshot_id}/restore — restaura in-place
		se.Router.POST("/x-api/services/{service_id}/snapshots/{snapshot_id}/restore", func(e *core.RequestEvent) error {
			autoBackup, err := manager.RestoreSnapshotInPlace(
				e.Request.Context(),
				e.Request.PathValue("service_id"),
				e.Request.PathValue("snapshot_id"),
			)
			if err != nil {
				return e.BadRequestError("failed to restore snapshot", err)
			}

			resp := map[string]any{"pre_restore_snapshot_id": nil}
			if autoBackup != nil {
				resp["pre_restore_snapshot_id"] = autoBackup.ID
				resp["pre_restore_snapshot_name"] = autoBackup.Name
			}
			return e.JSON(http.StatusOK, resp)
		}).Bind(apis.RequireAuth())

		// GET /x-api/services/{service_id}/snapshots/{snapshot_id}/download — descarga ZIP
		se.Router.GET("/x-api/services/{service_id}/snapshots/{snapshot_id}/download", func(e *core.RequestEvent) error {
			snapshot, err := manager.GetSnapshotFile(e.Request.Context(), e.Request.PathValue("service_id"), e.Request.PathValue("snapshot_id"))
			if err != nil {
				return e.BadRequestError("failed to find snapshot", err)
			}
			defer snapshot.Reader.Close()

			e.Response.Header().Set("Content-Type", "application/zip")
			e.Response.Header().Set("Content-Disposition", `attachment; filename="`+snapshot.Filename+`"`)

			if _, err := io.Copy(e.Response, snapshot.Reader); err != nil {
				return err
			}
			return nil
		}).Bind(apis.RequireAuth())

		// DELETE /x-api/services/{service_id}/snapshots/{snapshot_id} — elimina snapshot
		se.Router.DELETE("/x-api/services/{service_id}/snapshots/{snapshot_id}", func(e *core.RequestEvent) error {
			if err := manager.DeleteSnapshot(e.Request.Context(), e.Request.PathValue("service_id"), e.Request.PathValue("snapshot_id")); err != nil {
				return e.BadRequestError("failed to delete snapshot", err)
			}
			return e.NoContent(http.StatusNoContent)
		}).Bind(apis.RequireAuth())

		return se.Next()
	})
}

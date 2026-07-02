package hooks

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"pb_launcher/internal/launcher/domain"

	_ "modernc.org/sqlite"
)

const vacuumInterval = 3 * time.Hour

// RegisterSQLiteVacuum inicia un goroutine que cada 3 horas ejecuta VACUUM
// sobre los archivos SQLite (data.db y auxiliary.db) de todas las instancias
// que no tengan un proceso activo en memoria (sleeping, idle, stopped).
func RegisterSQLiteVacuum(app *pocketbase.PocketBase, lm *domain.LauncherManager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		go runVacuumLoop(app, lm)
		return se.Next()
	})
}

func runVacuumLoop(app *pocketbase.PocketBase, lm *domain.LauncherManager) {
	ticker := time.NewTicker(vacuumInterval)
	defer ticker.Stop()

	slog.Info("SQLite auto-vacuum iniciado", "interval", vacuumInterval.String())

	for range ticker.C {
		runVacuumSweep(app, lm)
	}
}

func runVacuumSweep(app *pocketbase.PocketBase, lm *domain.LauncherManager) {
	services, err := app.FindAllRecords("services")
	if err != nil {
		slog.Warn("sqlite-vacuum: no se pudieron obtener los servicios", "error", err)
		return
	}

	vacuumed, skipped, failed := 0, 0, 0

	for _, rec := range services {
		id := rec.Id

		// Saltar instancias con proceso activo en memoria — el archivo está bloqueado
		if lm.IsServiceRunning(id) {
			skipped++
			continue
		}

		serviceDataDir := filepath.Join(lm.DataDir(), id, "pb_data")

		for _, dbFile := range []string{"data.db", "auxiliary.db"} {
			fullPath := filepath.Join(serviceDataDir, dbFile)
			if _, statErr := os.Stat(fullPath); os.IsNotExist(statErr) {
				continue
			}
			if err := vacuumDB(fullPath); err != nil {
				slog.Warn("sqlite-vacuum: fallo en vacuum",
					"service", id,
					"file", dbFile,
					"error", err,
				)
				failed++
			} else {
				vacuumed++
			}
		}
	}

	slog.Info("sqlite-vacuum: barrido completado",
		"vacuumed_files", vacuumed,
		"skipped_running", skipped,
		"failed", failed,
	)
}

// vacuumDB abre el archivo SQLite indicado, hace checkpoint WAL y ejecuta
// VACUUM para compactar el espacio libre. Cierra la conexión inmediatamente.
func vacuumDB(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	// Un único hilo es suficiente — solo necesitamos una conexión exclusiva
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Asegurar que el WAL esté limpio antes de VACUUM
	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return err
	}

	if _, err := db.ExecContext(ctx, "VACUUM"); err != nil {
		return err
	}

	return nil
}

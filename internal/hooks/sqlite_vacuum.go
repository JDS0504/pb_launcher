package hooks

import (
	"context"
	"database/sql"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"pb_launcher/collections"
	"pb_launcher/internal/launcher/domain"

	_ "modernc.org/sqlite"
)

// zonaVacuum: UTC-5 fija (Perú), igual que operation_logs_cleanup.go.
var zonaVacuum = time.FixedZone("America/Lima", -5*60*60)

// Ventana activa: 02:00 (inclusive) → 05:00 (exclusive) hora Lima.
// Los barridos ocurren a las ~02:00, ~03:00 y ~04:00.
const ventanaInicio = 2
const ventanaFin = 5

// RegisterSQLiteVacuum inicia el planificador de vacuum SQLite.
// Solo opera dentro de la ventana nocturna 02:00–05:00 Lima.
// Prioriza las instancias con last_vacuum_at más antiguo (o nulo),
// garantizando cobertura equitativa aunque la ventana se cierre antes
// de completar todas las instancias.
func RegisterSQLiteVacuum(app *pocketbase.PocketBase, lm *domain.LauncherManager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		go runVacuumScheduler(app, lm)
		return se.Next()
	})
}

// runVacuumScheduler implementa el ciclo de vida del planificador:
//  1. Si estamos fuera de la ventana → esperar a las 02:00 Lima.
//  2. Dentro de la ventana → ejecutar barridos cada 1h + jitter (≤30 min).
//  3. Al salir de la ventana → volver al paso 1.
func runVacuumScheduler(app *pocketbase.PocketBase, lm *domain.LauncherManager) {
	for {
		if !enVentanaActiva() {
			dormirHastaVentana()
		}

		for enVentanaActiva() {
			runVacuumSweep(app, lm)

			// Jitter entre barridos: 1h + aleatorio ≤30 min.
			jitter := time.Duration(rand.Int63n(int64(30 * time.Minute)))
			slog.Info("sqlite-vacuum: próximo barrido en",
				"espera", (time.Hour + jitter).Round(time.Minute).String())
			time.Sleep(time.Hour + jitter)
		}
	}
}

// dormirHastaVentana duerme hasta las 02:00 Lima del día siguiente (o de hoy
// si aún no han llegado las 02:00).
func dormirHastaVentana() {
	ahora := time.Now().In(zonaVacuum)
	proximas2am := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), ventanaInicio, 0, 0, 0, zonaVacuum)
	if !ahora.Before(proximas2am) {
		proximas2am = proximas2am.Add(24 * time.Hour)
	}
	espera := time.Until(proximas2am)
	slog.Info("sqlite-vacuum: fuera de ventana nocturna, próximo ciclo en",
		"espera", espera.Round(time.Minute).String())
	time.Sleep(espera)
}

// enVentanaActiva devuelve true si la hora Lima actual está entre 02:00 y 05:00.
func enVentanaActiva() bool {
	hora := time.Now().In(zonaVacuum).Hour()
	return hora >= ventanaInicio && hora < ventanaFin
}

// runVacuumSweep obtiene los servicios ordenados por last_vacuum_at ASC NULLS FIRST:
// primero los que nunca se vacunaron, luego los más antiguos.
// Esto garantiza que, si la ventana se cierra a mitad del barrido, siempre
// se hayan cubierto las instancias con mayor deuda de vacuum.
func runVacuumSweep(app *pocketbase.PocketBase, lm *domain.LauncherManager) {
	// ORDER BY last_vacuum_at ASC: NULL primero (nunca vacunados), luego los más antiguos.
	services, err := app.FindAllRecords(collections.Services)
	if err != nil {
		slog.Warn("sqlite-vacuum: no se pudieron obtener los servicios", "error", err)
		return
	}

	// Ordenar en Go: NULL (vacío) primero, luego por fecha ascendente.
	sortByLastVacuum(services)

	slog.Info("sqlite-vacuum: iniciando barrido", "instancias", len(services))
	vacuumed, skipped, failed := 0, 0, 0

	for _, rec := range services {
		// Hard-stop: salir de la ventana 02:00–05:00 → detener el barrido.
		// Las instancias restantes ya están ordenadas para ser las primeras
		// del siguiente barrido nocturno (tienen last_vacuum_at más reciente).
		if !enVentanaActiva() {
			slog.Info("sqlite-vacuum: ventana cerrada, deteniendo barrido",
				"vacuumed", vacuumed, "pending", len(services)-vacuumed-skipped-failed)
			break
		}

		id := rec.Id

		if lm.IsServiceRunning(id) {
			skipped++
			continue
		}

		// Jitter (≤10 s): respiro mínimo de I/O entre instancias consecutivas.
		time.Sleep(time.Duration(rand.Int63n(int64(10 * time.Second))))

		// Re-verificar tras el jitter (pudo despertar durante la pausa).
		if lm.IsServiceRunning(id) {
			skipped++
			continue
		}

		// Marcar como "en vacuum" para que WakeupService espere antes de
		// arrancar PocketBase y evitar SQLITE_BUSY.
		lm.LockVacuum(id)

		name := rec.GetString("name")
		serviceDataDir := filepath.Join(lm.DataDir(), name, "pb_data")
		allOk := true
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
				allOk = false
			} else {
				vacuumed++
			}
		}

		lm.UnlockVacuum(id)

		// Actualizar last_vacuum_at solo si todos los archivos se compactaron bien.
		if allOk {
			if err := markVacuumDone(app, id); err != nil {
				slog.Warn("sqlite-vacuum: no se pudo actualizar last_vacuum_at",
					"service", id, "error", err)
			}
		}
	}

	slog.Info("sqlite-vacuum: barrido completado",
		"vacuumed_files", vacuumed,
		"skipped_running", skipped,
		"failed", failed,
	)
}

// sortByLastVacuum ordena in-place: primero los registros con last_vacuum_at
// vacío o nulo (nunca vacunados), luego por fecha ascendente (más antiguos primero).
func sortByLastVacuum(records []*core.Record) {
	for i := 1; i < len(records); i++ {
		for j := i; j > 0; j-- {
			a := records[j-1].GetDateTime("last_vacuum_at")
			b := records[j].GetDateTime("last_vacuum_at")
			aZero := a.IsZero()
			bZero := b.IsZero()
			// NULL/vacío siempre va primero; si ambos tienen fecha, el más antiguo va primero.
			if (!aZero && bZero) || (!aZero && !bZero && a.Time().After(b.Time())) {
				records[j-1], records[j] = records[j], records[j-1]
			} else {
				break
			}
		}
	}
}

// markVacuumDone escribe la hora actual en last_vacuum_at del servicio.
func markVacuumDone(app *pocketbase.PocketBase, serviceID string) error {
	record, err := app.FindRecordById(collections.Services, serviceID)
	if err != nil {
		return err
	}
	record.Set("last_vacuum_at", time.Now().UTC())
	return app.Save(record)
}

// vacuumDB abre el archivo SQLite, hace checkpoint WAL y ejecuta VACUUM.
//
// Duty-cycle 50 %: después de cada VACUUM duerme el mismo tiempo que tardó
// la operación para no consumir más del 50 % de un núcleo de forma sostenida.
func vacuumDB(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return err
	}

	inicio := time.Now()
	if _, err := db.ExecContext(ctx, "VACUUM"); err != nil {
		return err
	}
	elapsed := time.Since(inicio)

	// Duty-cycle 50%: dormir lo mismo que tardó → CPU promedio ≤ 50% por archivo.
	if elapsed > 0 {
		time.Sleep(elapsed)
	}

	return nil
}

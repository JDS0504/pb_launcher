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
// Solo opera dentro de la ventana nocturna 00:00–05:00 Lima.
// Cada barrido se repite cada 1h + jitter aleatorio (≤30 min) para no
// generar un "thundering herd" de lecturas/escrituras simultáneas.
// Por cada instancia inactiva aplica un jitter adicional (≤5 min) antes
// de vacunar sus archivos, dispersando aún más la carga.
// El duty-cycle del VACUUM se limita al 50 %: si compactar un archivo
// tarda T segundos, el proceso duerme T segundos antes de continuar.
func RegisterSQLiteVacuum(app *pocketbase.PocketBase, lm *domain.LauncherManager) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		go runVacuumScheduler(app, lm)
		return se.Next()
	})
}

// runVacuumScheduler implementa el ciclo de vida del planificador:
//  1. Si estamos fuera de la ventana → esperar a la próxima medianoche Lima.
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
			// Si el resultado excede las 05:00, el bucle externo lo detecta y rompe.
			jitter := time.Duration(rand.Int63n(int64(30 * time.Minute)))
			slog.Info("sqlite-vacuum: próximo barrido en",
				"espera", (time.Hour + jitter).Round(time.Minute).String())
			time.Sleep(time.Hour + jitter)
		}
	}
}

// dormirHastaVentana calcula el tiempo restante hasta las 02:00 Lima
// del día siguiente y bloquea el goroutine ese tiempo.
func dormirHastaVentana() {
	ahora := time.Now().In(zonaVacuum)
	proximas2am := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), ventanaInicio, 0, 0, 0, zonaVacuum)
	if !ahora.Before(proximas2am) {
		// Ya pasaron las 02:00 de hoy → apuntar a mañana
		proximas2am = proximas2am.Add(24 * time.Hour)
	}
	espera := time.Until(proximas2am)
	slog.Info("sqlite-vacuum: fuera de ventana nocturna, próximo ciclo en",
		"espera", espera.Round(time.Minute).String())
	time.Sleep(espera)
}

// enVentanaActiva devuelve true si la hora Lima actual está entre 00:00 y 05:00.
func enVentanaActiva() bool {
	hora := time.Now().In(zonaVacuum).Hour()
	return hora >= ventanaInicio && hora < ventanaFin
}

// runVacuumSweep itera todos los servicios registrados y compacta sus
// archivos SQLite si la instancia no tiene un proceso activo en memoria.
func runVacuumSweep(app *pocketbase.PocketBase, lm *domain.LauncherManager) {
	services, err := app.FindAllRecords("services")
	if err != nil {
		slog.Warn("sqlite-vacuum: no se pudieron obtener los servicios", "error", err)
		return
	}

	slog.Info("sqlite-vacuum: iniciando barrido", "instancias", len(services))
	vacuumed, skipped, failed := 0, 0, 0

	for _, rec := range services {
		id := rec.Id

		// Primera comprobación: proceso activo → archivo bloqueado, saltar.
		if lm.IsServiceRunning(id) {
			skipped++
			continue
		}

		// Jitter por servicio (≤5 min): evita que todas las instancias se
		// compacten en el mismo segundo (thundering herd de I/O).
		jitter := time.Duration(rand.Int63n(int64(5 * time.Minute)))
		time.Sleep(jitter)

		// Segunda comprobación tras el jitter: la instancia puede haber
		// despertado durante ese intervalo.
		if lm.IsServiceRunning(id) {
			skipped++
			continue
		}

		// Marcar como "en vacuum" para que WakeupService espere antes de
		// arrancar PocketBase y evitar SQLITE_BUSY.
		lm.LockVacuum(id)

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

		lm.UnlockVacuum(id)
	}

	slog.Info("sqlite-vacuum: barrido completado",
		"vacuumed_files", vacuumed,
		"skipped_running", skipped,
		"failed", failed,
	)
}

// vacuumDB abre el archivo SQLite, hace checkpoint WAL y ejecuta VACUUM.
//
// Duty-cycle 50 %: después de cada VACUUM duerme el mismo tiempo que tardó
// la operación. Así el proceso nunca consume más del 50 % de un núcleo de
// forma sostenida, dejando capacidad libre para el resto del sistema.
func vacuumDB(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	// Conexión exclusiva mínima: solo necesitamos un hilo para el VACUUM.
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Fusionar el WAL pendiente al archivo principal antes de compactar.
	if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return err
	}

	// Medir duración del VACUUM para calcular el tiempo de reposo (duty-cycle 50 %).
	inicio := time.Now()
	if _, err := db.ExecContext(ctx, "VACUUM"); err != nil {
		return err
	}
	elapsed := time.Since(inicio)

	// Reposo igual al tiempo del VACUUM → CPU promedio ≤ 50 % por archivo.
	if elapsed > 0 {
		time.Sleep(elapsed)
	}

	return nil
}

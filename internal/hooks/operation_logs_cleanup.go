package hooks

import (
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// zonaLima es UTC-5 fija (Perú no aplica horario de verano).
var zonaLima = time.FixedZone("America/Lima", -5*60*60)

// RegisterOperationLogsCleanup inicia un goroutine en background que elimina
// en batch los registros de operation_logs con más de 7 días de antigüedad.
// La limpieza se ejecuta todos los días a las 04:00 hora Lima (UTC-5).
func RegisterOperationLogsCleanup(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		go func() {
			purge := func() {
				// Eliminar operation_logs con más de 7 días
				_, _ = app.DB().NewQuery(
					`DELETE FROM operation_logs WHERE created < datetime('now', '-7 days')`,
				).Execute()
				// Eliminar comands completados (success/error) con más de 7 días.
				// Los pending se conservan indefinidamente hasta ser ejecutados.
				_, _ = app.DB().NewQuery(
					`DELETE FROM comands WHERE status != 'pending' AND created < datetime('now', '-7 days')`,
				).Execute()
			}

			for {
				ahora := time.Now().In(zonaLima)

				// Próxima 04:00 hora Lima
				proxima := time.Date(
					ahora.Year(), ahora.Month(), ahora.Day(),
					4, 0, 0, 0,
					zonaLima,
				)
				// Si ya pasó la 01:00 de hoy, apuntar al día siguiente
				if !ahora.Before(proxima) {
					proxima = proxima.Add(24 * time.Hour)
				}

				espera := time.Until(proxima)
				time.Sleep(espera)
				purge()
			}
		}()
		return se.Next()
	})
}

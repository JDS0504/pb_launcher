package migrations

import (
	"pb_launcher/collections"

	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/core"
)

// Activa CascadeDelete en los campos FK `service` de services_domains, comands
// y operation_logs modificando el JSON almacenado en _collections directamente.
// Esto bypasa la validación Go de "System fields cannot be changed" que bloquea
// app.Save(), pero PocketBase lee el JSON correcto al arrancar y gestiona el
// cascade automáticamente a nivel aplicación.
func init() {
	setCascade := func(app core.App, collectionName string, enable bool) error {
		val := "true"
		if !enable {
			val = "false"
		}
		_, err := app.DB().NewQuery(`
			UPDATE _collections
			SET fields = (
				SELECT json_group_array(
					CASE
						WHEN json_extract(f.value, '$.name') = 'service'
						THEN json_set(f.value, '$.cascadeDelete', json(` + "'" + val + "'" + `))
						ELSE f.value
					END
				)
				FROM json_each(fields) AS f
			)
			WHERE name = {:collection}
		`).Bind(map[string]any{"collection": collectionName}).Execute()
		return err
	}

	m.Register(func(app core.App) error {
		if err := setCascade(app, collections.ServicesDomains, true); err != nil {
			return err
		}
		if err := setCascade(app, collections.ServicesComands, true); err != nil {
			return err
		}
		return setCascade(app, collections.OperationLogs, true)
	}, func(app core.App) error {
		if err := setCascade(app, collections.ServicesDomains, false); err != nil {
			return err
		}
		if err := setCascade(app, collections.ServicesComands, false); err != nil {
			return err
		}
		return setCascade(app, collections.OperationLogs, false)
	})
}

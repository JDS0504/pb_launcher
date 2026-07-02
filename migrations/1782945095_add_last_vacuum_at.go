package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}

		// Registra cuándo se ejecutó el último VACUUM SQLite sobre los archivos
		// de la instancia. El planificador nocturno lo usa para priorizar las
		// instancias que llevan más tiempo sin compactar (ORDER BY ASC NULLS FIRST).
		services.Fields.Add(
			&core.DateField{
				Name:   "last_vacuum_at",
				System: true,
			},
		)

		return app.Save(services)
	}, func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}

		services.Fields.RemoveByName("last_vacuum_at")

		return app.Save(services)
	})
}

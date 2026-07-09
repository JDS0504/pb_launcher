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

		// Agregar campo current_snapshot_applied_at a la colección de servicios
		services.Fields.Add(
			&core.DateField{
				Name:   "current_snapshot_applied_at",
				System: true,
			},
		)
		return app.Save(services)
	}, func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err == nil {
			services.Fields.RemoveByName("current_snapshot_applied_at")
			_ = app.Save(services)
		}
		return nil
	})
}

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

		// Como 'deleted' es un campo de sistema (System: true), PocketBase no permite
		// eliminarlo directamente. Primero debemos cambiar 'System' a false, guardar la
		// colección, y luego eliminarlo en un segundo paso.
		if f, ok := services.Fields.GetByName("deleted").(*core.DateField); ok {
			if f.System {
				f.System = false
				if err := app.Save(services); err != nil {
					return err
				}
			}
		}

		services.Fields.RemoveByName("deleted")
		return app.Save(services)
	}, func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}

		// Revertir: Volver a agregar el campo deleted al esquema
		services.Fields.Add(
			&core.DateField{
				Name:   "deleted",
				System: true,
			},
		)

		return app.Save(services)
	})
}

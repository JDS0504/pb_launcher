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

		// Elimina el campo heredado 'deleted' de la colección services, ya que
		// el borrado ahora es físico (hard delete) y no lógico (soft delete).
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

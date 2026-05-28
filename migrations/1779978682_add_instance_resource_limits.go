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
		
		// Agregar los nuevos campos a la colección
		services.Fields.Add(
			&core.TextField{
				Name: "cpu_quota",
			},
			&core.TextField{
				Name: "memory_limit",
			},
		)
		
		return app.Save(services)
	}, func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		
		// Revertir los campos agregados
		services.Fields.RemoveByName("cpu_quota")
		services.Fields.RemoveByName("memory_limit")
		
		return app.Save(services)
	})
}

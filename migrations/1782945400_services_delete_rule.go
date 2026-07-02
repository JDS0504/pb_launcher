package migrations

import (
	"pb_launcher/collections"
	"pb_launcher/utils"

	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/core"
)

// Habilita DeleteRule en la colección services para que cualquier usuario
// autenticado pueda eliminar instancias (hard-delete).
// Sin esta regla PocketBase rechaza el DELETE con "Only superusers can perform this action".
func init() {
	m.Register(func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		services.DeleteRule = utils.StrPointer(`@request.auth.id != ""`)
		return app.Save(services)
	}, func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		services.DeleteRule = nil
		return app.Save(services)
	})
}

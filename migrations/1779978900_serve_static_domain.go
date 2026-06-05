package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}

		// Campo booleano: cuando es true, el proxy sirve pb_public desde disco
		// sin necesidad de encender la instancia de PocketBase.
		// Solo aplica cuando el dominio está asociado a un servicio (service != "").
		col.Fields.Add(&core.BoolField{
			Name:   "serve_static",
			System: true,
		})

		return app.Save(col)
	}, func(app core.App) error {
		col, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}
		col.Fields.RemoveByName("serve_static")
		return app.Save(col)
	})
}

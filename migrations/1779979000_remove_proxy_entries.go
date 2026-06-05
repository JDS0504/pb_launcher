package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Eliminar proxy_entry y serve_static de services_domains
		col, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}
		col.Fields.RemoveByName("proxy_entry")
		col.Fields.RemoveByName("serve_static")
		if err := app.Save(col); err != nil {
			return err
		}

		// 2. Eliminar la colección proxy_entries si existe
		proxyEntries, err := app.FindCollectionByNameOrId(collections.ProxyEntries)
		if err != nil {
			// Si no existe, no hay nada que borrar
			return nil
		}
		return app.Delete(proxyEntries)
	}, func(app core.App) error {
		// Rollback: no se puede recuperar datos eliminados, solo recrear estructura vacía
		return nil
	})
}

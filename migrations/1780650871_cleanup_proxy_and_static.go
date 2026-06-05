package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Modificar services_domains
		servicesDomains, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}

		// Desmarcar como campos de sistema para poder eliminarlos
		for _, f := range servicesDomains.Fields {
			if f.GetName() == "serve_static" || f.GetName() == "proxy_entry" {
				if bf, ok := f.(*core.BoolField); ok {
					bf.System = false
				}
				if rf, ok := f.(*core.RelationField); ok {
					rf.System = false
				}
			}
		}

		// Remover campos proxy_entry y serve_static
		servicesDomains.Fields.RemoveByName("proxy_entry")
		servicesDomains.Fields.RemoveByName("serve_static")

		// Hacer que el campo service sea requerido
		serviceField := servicesDomains.Fields.GetByName("service")
		if rf, ok := serviceField.(*core.RelationField); ok {
			rf.Required = true
		}

		if err := app.Save(servicesDomains); err != nil {
			return err
		}

		// 2. Eliminar la coleccion proxy_entries
		proxyEntries, err := app.FindCollectionByNameOrId("proxy_entries")
		if err == nil && proxyEntries != nil {
			if err := app.Delete(proxyEntries); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		return nil
	})
}

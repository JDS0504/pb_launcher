package migrations

import (
	"pb_launcher/collections"
	"pb_launcher/utils"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Aplica la regla `@request.auth.id != ""` para lectura, edición, creación
// y eliminación a todas las colecciones principales del sistema.
// Esto asegura que ninguna operación se pueda realizar sin autenticación.
func init() {
	m.Register(func(app core.App) error {
		cols := []string{
			collections.Repositories,
			collections.Releases,
			collections.Services,
			collections.ServicesDomains,
			collections.ServicesComands,
			collections.CertRequests,
			collections.OperationLogs,
			collections.ProxyEntries,
		}

		rule := utils.StrPointer(`@request.auth.id != ""`)

		for _, name := range cols {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				app.Logger().Warn("Collection not found for auth rule update", "collection", name)
				continue
			}

			collection.ListRule = rule
			collection.ViewRule = rule
			collection.CreateRule = rule
			collection.UpdateRule = rule
			collection.DeleteRule = rule

			if err := app.Save(collection); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// El downgrade explícito para revertir estas reglas dependería del estado original
		// de cada colección. Como es una política de seguridad general,
		// un rollback genérico que ponga todo a 'nil' (admin-only) es lo más seguro
		// si se revierte esta migración.

		cols := []string{
			collections.Repositories,
			collections.Releases,
			collections.Services,
			collections.ServicesDomains,
			collections.ServicesComands,
			collections.CertRequests,
			collections.OperationLogs,
			collections.ProxyEntries,
		}

		for _, name := range cols {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				continue // Ignorar si no existe en el downgrade
			}
			
			collection.ListRule = nil
			collection.ViewRule = nil
			collection.CreateRule = nil
			collection.UpdateRule = nil
			collection.DeleteRule = nil

			_ = app.Save(collection)
		}

		return nil
	})
}

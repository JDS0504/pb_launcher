package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migración que:
//  1. Limpia del disco los servicios que quedaron en soft-delete (deleted != "")
//     usando SQL directo (bypassa hooks y FK checks de PocketBase).
//  2. Elimina el campo `deleted` de services: ya no hay soft-delete; la
//     eliminación es siempre física (hard-delete via API DELETE).
//
// NOTA: CascadeDelete en los campos FK (service) NO se aplica aquí porque
// esos campos tienen System: true y PocketBase no permite modificarlos.
// El cascade se maneja explícitamente en el hook OnRecordDeleteRequest
// de services.go usando DELETE SQL directo antes de borrar el registro padre.
func init() {
	m.Register(func(app core.App) error {
		db := app.DB()

		// ── 1. Limpiar registros relacionados de servicios soft-deleted ────────
		// Primero los hijos (FK apuntan a services), luego el padre.
		_, _ = db.NewQuery(`
			DELETE FROM services_domains WHERE service IN (
				SELECT id FROM services WHERE deleted IS NOT NULL AND deleted != ''
			)
		`).Execute()
		_, _ = db.NewQuery(`
			DELETE FROM comands WHERE service IN (
				SELECT id FROM services WHERE deleted IS NOT NULL AND deleted != ''
			)
		`).Execute()
		_, _ = db.NewQuery(`
			DELETE FROM operation_logs WHERE service IN (
				SELECT id FROM services WHERE deleted IS NOT NULL AND deleted != ''
			)
		`).Execute()

		// ── 2. Borrar los servicios soft-deleted ──────────────────────────────
		_, _ = db.NewQuery(`
			DELETE FROM services WHERE deleted IS NOT NULL AND deleted != ''
		`).Execute()

		// ── 3. Eliminar campo `deleted` del schema de services ────────────────
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		services.Fields.RemoveByName("deleted")
		return app.Save(services)
	}, func(app core.App) error {
		// Downgrade: restaurar el campo `deleted`
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		services.Fields.Add(&core.DateField{
			Name:   "deleted",
			System: true,
		})
		return app.Save(services)
	})
}

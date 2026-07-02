package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migración que:
//  1. Limpia servicios en soft-delete previos (limpia hijos primero via SQL directo).
//  2. Elimina el campo `deleted` de services (ya no hay soft-delete).
//
// NOTA: CascadeDelete en campos FK con System:true no puede modificarse
// via app.Save(). El cascade se gestiona explícitamente en el hook
// OnRecordDeleteRequest de services.go (3 DELETE SQL antes del e.Next()).
func init() {
	m.Register(func(app core.App) error {
		db := app.DB()

		// ── 1. Limpiar hijos de servicios soft-deleted ─────────────────────────
		// SQL directo: bypassa hooks y FK checks. Primero hijos, luego padre.
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

		// ── 3. Eliminar campo `deleted` del schema ────────────────────────────
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		services.Fields.RemoveByName("deleted")
		return app.Save(services)
	}, func(app core.App) error {
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

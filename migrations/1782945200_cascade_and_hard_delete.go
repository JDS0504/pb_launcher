package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migración que:
//  1. Activa CascadeDelete en los campos FK `service` de services_domains,
//     comands y operation_logs. Al eliminar un servicio, PocketBase borra
//     automáticamente todos sus hijos — no se necesita código extra.
//     TRUCO: se pone System=false antes de guardar para pasar la validación
//     (PocketBase salta el check de system-fields para campos no-system).
//  2. Limpia servicios en soft-delete previos con SQL directo.
//  3. Elimina el campo `deleted` de services (ya no hay soft-delete).
func init() {
	m.Register(func(app core.App) error {
		// ── 1. CascadeDelete en services_domains.service ──────────────────────
		sd, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}
		if f, ok := sd.Fields.GetByName("service").(*core.RelationField); ok {
			f.System = false // desactiva el check de field inmutable
			f.CascadeDelete = true
		}
		if err := app.Save(sd); err != nil {
			return err
		}

		// ── 2. CascadeDelete en comands.service ───────────────────────────────
		comands, err := app.FindCollectionByNameOrId(collections.ServicesComands)
		if err != nil {
			return err
		}
		if f, ok := comands.Fields.GetByName("service").(*core.RelationField); ok {
			f.System = false
			f.CascadeDelete = true
		}
		if err := app.Save(comands); err != nil {
			return err
		}

		// ── 3. CascadeDelete en operation_logs.service ────────────────────────
		logs, err := app.FindCollectionByNameOrId(collections.OperationLogs)
		if err != nil {
			return err
		}
		if f, ok := logs.Fields.GetByName("service").(*core.RelationField); ok {
			f.System = false
			f.CascadeDelete = true
		}
		if err := app.Save(logs); err != nil {
			return err
		}

		// ── 4. Limpiar servicios soft-deleted existentes ──────────────────────
		// SQL directo: bypassa hooks y FK checks. Los hijos ya no existen
		// (CascadeDelete activo) así que se pueden borrar directamente.
		db := app.DB()
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
		_, _ = db.NewQuery(`
			DELETE FROM services WHERE deleted IS NOT NULL AND deleted != ''
		`).Execute()

		// ── 5. Eliminar campo `deleted` del schema de services ────────────────
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		services.Fields.RemoveByName("deleted")
		return app.Save(services)
	}, func(app core.App) error {
		// Downgrade: revertir CascadeDelete y restaurar campo deleted
		sd, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}
		if f, ok := sd.Fields.GetByName("service").(*core.RelationField); ok {
			f.CascadeDelete = false
		}
		if err := app.Save(sd); err != nil {
			return err
		}

		comands, err := app.FindCollectionByNameOrId(collections.ServicesComands)
		if err != nil {
			return err
		}
		if f, ok := comands.Fields.GetByName("service").(*core.RelationField); ok {
			f.CascadeDelete = false
		}
		if err := app.Save(comands); err != nil {
			return err
		}

		logs, err := app.FindCollectionByNameOrId(collections.OperationLogs)
		if err != nil {
			return err
		}
		if f, ok := logs.Fields.GetByName("service").(*core.RelationField); ok {
			f.CascadeDelete = false
		}
		if err := app.Save(logs); err != nil {
			return err
		}

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

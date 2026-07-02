package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migración que:
//  1. Añade CascadeDelete en los FK de service en services_domains, comands y
//     operation_logs: al borrar un servicio PocketBase elimina automáticamente
//     todos los registros relacionados (SSOT, sin código Go adicional).
//  2. Limpia del disco los servicios que quedaron en soft-delete (deleted != "")
//     bajo el mecanismo anterior.
//  3. Elimina el campo `deleted` de services: ya no hay soft-delete; la
//     eliminación es siempre física (hard-delete via API DELETE).
func init() {
	m.Register(func(app core.App) error {
		// ── 1. CascadeDelete en services_domains.service ──────────────────────
		sd, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}
		if f, ok := sd.Fields.GetByName("service").(*core.RelationField); ok {
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
			f.CascadeDelete = true
		}
		if err := app.Save(logs); err != nil {
			return err
		}

		// ── 4. Hard-delete de servicios en soft-delete ────────────────────────
		// Los relacionados (services_domains, comands, operation_logs) se eliminan
		// automáticamente gracias al CascadeDelete ya activado arriba.
		softDeleted, err := app.FindAllRecords(
			collections.Services,
			// Filtramos los que tienen deleted distinto de vacío
		)
		if err == nil {
			for _, rec := range softDeleted {
				if d := rec.GetDateTime("deleted"); !d.IsZero() {
					_ = app.Delete(rec)
				}
			}
		}

		// ── 5. Eliminar campo `deleted` de services ───────────────────────────
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		services.Fields.RemoveByName("deleted")
		return app.Save(services)
	}, func(app core.App) error {
		// ── Downgrade: restaurar CascadeDelete = false y re-añadir deleted ────
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

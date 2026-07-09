package migrations

import (
	"pb_launcher/collections"
	"pb_launcher/utils"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}

		// 1. Crear colección service_snapshots
		snapshots := core.NewBaseCollection(collections.ServiceSnapshots)
		snapshots.Fields.Add(
			&core.RelationField{
				Name:         "service",
				CollectionId: services.Id,
				System:       true,
				Required:     true,
				MinSelect:    1,
				MaxSelect:    1,
			},
			&core.TextField{
				Name:     "name",
				System:   true,
				Required: true,
			},
			&core.TextField{
				Name:   "comment",
				System: true,
			},
			&core.SelectField{
				Name:      "type",
				System:    true,
				Required:  true,
				MaxSelect: 1,
				Values:    []string{"manual", "pre-restore"},
			},
			&core.TextField{
				Name:   "version",
				System: true,
			},
			&core.FileField{
				Name:      "file",
				System:    true,
				MaxSelect: 1,
			},
			&core.NumberField{
				Name:   "size",
				System: true,
			},
			&core.AutodateField{
				Name:     "created",
				OnCreate: true,
				System:   true,
			},
		)
		snapshots.Indexes = append(snapshots.Indexes,
			`CREATE INDEX idx_service_snapshots_service ON service_snapshots(service)`,
			`CREATE INDEX idx_service_snapshots_created ON service_snapshots(created)`,
		)
		snapshots.ListRule = utils.StrPointer(`@request.auth.id != ""`)
		snapshots.ViewRule = utils.StrPointer(`@request.auth.id != ""`)
		snapshots.CreateRule = utils.StrPointer(`@request.auth.id != ""`)
		snapshots.UpdateRule = utils.StrPointer(`@request.auth.id != ""`)
		snapshots.DeleteRule = utils.StrPointer(`@request.auth.id != ""`)

		if err := app.Save(snapshots); err != nil {
			return err
		}

		// 2. Agregar campo current_snapshot_id a services
		services.Fields.Add(
			&core.TextField{
				Name:   "current_snapshot_id",
				System: true,
			},
		)
		return app.Save(services)
	}, func(app core.App) error {
		// Revertir: eliminar current_snapshot_id de services
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err == nil {
			services.Fields.RemoveByName("current_snapshot_id")
			_ = app.Save(services)
		}
		// Eliminar colección service_snapshots
		col, err := app.FindCollectionByNameOrId(collections.ServiceSnapshots)
		if err != nil {
			return nil
		}
		return app.Delete(col)
	})
}

package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		repositories, err := app.FindCollectionByNameOrId(collections.Repositories)
		if err != nil {
			return err
		}
		repositories.Fields.Add(
			&core.DateField{Name: "last_sync_at", System: true},
			&core.SelectField{
				Name:      "last_sync_status",
				System:    true,
				MaxSelect: 1,
				Values:    []string{"never", "syncing", "success", "error"},
			},
			&core.TextField{Name: "last_sync_error", System: true},
		)
		return app.Save(repositories)
	}, func(app core.App) error {
		repositories, err := app.FindCollectionByNameOrId(collections.Repositories)
		if err != nil {
			return err
		}
		repositories.Fields.RemoveByName("last_sync_at")
		repositories.Fields.RemoveByName("last_sync_status")
		repositories.Fields.RemoveByName("last_sync_error")
		return app.Save(repositories)
	})
}

package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// Remove repository field from releases collection
		releases, err := app.FindCollectionByNameOrId(collections.Releases)
		if err != nil {
			return err
		}

		releases.Fields.RemoveByName("repository")

		// Update unique index for releases to just (version) instead of (repository,version)
		releases.Indexes = []string{"CREATE UNIQUE INDEX idx_releases ON releases (version)"}

		if err := app.Save(releases); err != nil {
			return err
		}

		// Delete repositories collection
		repositories, err := app.FindCollectionByNameOrId(collections.Repositories)
		if err == nil {
			if err := app.Delete(repositories); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		// Downgrade not implemented for deleting a core architecture table
		return nil
	})
}

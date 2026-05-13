package migrations

import (
	"pb_launcher/collections"
	"pb_launcher/utils"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		repositories, err := app.FindCollectionByNameOrId(collections.Repositories)
		if err != nil {
			return err
		}
		repositories.CreateRule = utils.StrPointer(`@request.auth.id != ""`)
		repositories.UpdateRule = utils.StrPointer(`@request.auth.id != ""`)
		repositories.DeleteRule = utils.StrPointer(`@request.auth.id != ""`)
		return app.Save(repositories)
	}, func(app core.App) error {
		repositories, err := app.FindCollectionByNameOrId(collections.Repositories)
		if err != nil {
			return err
		}
		repositories.CreateRule = nil
		repositories.UpdateRule = nil
		repositories.DeleteRule = nil
		return app.Save(repositories)
	})
}

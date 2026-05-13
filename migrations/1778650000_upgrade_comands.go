package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		comands, err := app.FindCollectionByNameOrId(collections.ServicesComands)
		if err != nil {
			return err
		}
		releases, err := app.FindCollectionByNameOrId(collections.Releases)
		if err != nil {
			return err
		}

		if actionField, ok := comands.Fields.GetByName("action").(*core.SelectField); ok {
			actionField.Values = append(actionField.Values, "upgrade")
		}

		comands.Fields.Add(&core.RelationField{
			Name:         "target_release",
			CollectionId: releases.Id,
			System:       true,
			MinSelect:    0,
			MaxSelect:    1,
		})

		return app.Save(comands)
	}, func(app core.App) error {
		comands, err := app.FindCollectionByNameOrId(collections.ServicesComands)
		if err != nil {
			return err
		}

		if actionField, ok := comands.Fields.GetByName("action").(*core.SelectField); ok {
			values := actionField.Values[:0]
			for _, value := range actionField.Values {
				if value != "upgrade" {
					values = append(values, value)
				}
			}
			actionField.Values = values
		}
		comands.Fields.RemoveByName("target_release")
		return app.Save(comands)
	})
}

package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		if statusField, ok := services.Fields.GetByName("status").(*core.SelectField); ok {
			statusField.Values = append(statusField.Values, "restoring")
		}
		return app.Save(services)
	}, func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}
		if statusField, ok := services.Fields.GetByName("status").(*core.SelectField); ok {
			values := statusField.Values[:0]
			for _, value := range statusField.Values {
				if value != "restoring" {
					values = append(values, value)
				}
			}
			statusField.Values = values
		}
		return app.Save(services)
	})
}

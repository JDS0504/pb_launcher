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
			exists := false
			for _, v := range statusField.Values {
				if v == "sleeping" {
					exists = true
					break
				}
			}
			if !exists {
				statusField.Values = append(statusField.Values, "sleeping")
			}
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
				if value != "sleeping" {
					values = append(values, value)
				}
			}
			statusField.Values = values
		}
		return app.Save(services)
	})
}

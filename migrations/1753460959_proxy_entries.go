package migrations

import (
	"pb_launcher/collections"
	"pb_launcher/utils"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		proxyEntries := core.NewBaseCollection("proxy_entries")
		proxyEntries.Fields.Add(
			&core.TextField{
				Name:        "name",
				System:      true,
				Required:    true,
				Presentable: true,
			},
			&core.URLField{
				Name:     "target_url",
				System:   true,
				Required: true,
			},
			&core.SelectField{
				Name:      "enabled",
				System:    true,
				MaxSelect: 1,
				Values:    []string{"yes", "no"},
				Required:  true,
			},
			&core.DateField{
				Name:   "deleted",
				System: true,
			},
		)
		proxyEntries.ListRule = utils.StrPointer(`@request.auth.id != ""`)
		proxyEntries.ViewRule = utils.StrPointer(`@request.auth.id != ""`)
		proxyEntries.CreateRule = utils.StrPointer(`@request.auth.id != ""`)
		proxyEntries.UpdateRule = utils.StrPointer(`@request.auth.id != ""`)

		if err := app.Save(proxyEntries); err != nil {
			return err
		}

		servicesDomains, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}

		serviceField := servicesDomains.Fields.GetByName("service")
		if rf, ok := serviceField.(*core.RelationField); ok {
			rf.Required = false
		}

		servicesDomains.Fields.AddAt(3, &core.RelationField{
			Name:         "proxy_entry",
			CollectionId: proxyEntries.Id,
		})

		return app.Save(servicesDomains)
	}, func(app core.App) error {
		servicesDomains, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}
		servicesDomains.Fields.RemoveByName("proxy_entry")
		if err := app.Save(servicesDomains); err != nil {
			return err
		}
		proxyEntries, err := app.FindCollectionByNameOrId("proxy_entries")
		if err != nil {
			return err
		}
		return app.Delete(proxyEntries)
	})
}

package migrations

import (
	"pb_launcher/utils"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		certRequests := core.NewBaseCollection("cert_requests")
		certRequests.Fields.Add(
			&core.TextField{
				Name:     "domain",
				System:   true,
				Required: true,
				Pattern:  DomainPattern,
			},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"pending", "approved", "failed"},
				MaxSelect: 1,
				System:    true,
				Required:  true,
			},
			&core.DateField{
				Name:   "not_before",
				System: true,
			},
			&core.NumberField{
				Name:     "attempt",
				Min:      utils.Ptr[float64](1),
				System:   true,
				OnlyInt:  true,
				Required: true,
			},
			&core.TextField{
				Name:   "message",
				System: true,
			},
			&core.DateField{
				Name:   "requested",
				System: true,
			},
			&core.AutodateField{
				Name:     "created",
				System:   true,
				OnCreate: true,
			},
		)
		certRequests.CreateRule = utils.StrPointer(`@request.auth.id != ""`)

		return app.Save(certRequests)
	}, func(app core.App) error {
		certRequests, err := app.FindCollectionByNameOrId("cert_requests")
		if err != nil {
			return err
		}
		return app.Delete(certRequests)
	})
}

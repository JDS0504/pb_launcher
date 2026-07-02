package migrations

import (
	"pb_launcher/collections"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// 1. Añadir campos de certificado a services_domains
		domains, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}

		domains.Fields.Add(&core.SelectField{
			Name:      "cert_status",
			Values:    []string{"pending", "approved", "failed"},
			MaxSelect: 1,
			System:    true,
		})
		domains.Fields.Add(&core.DateField{
			Name:   "cert_not_before",
			System: true,
		})
		domains.Fields.Add(&core.DateField{
			Name:   "cert_not_after",
			System: true,
		})
		domains.Fields.Add(&core.TextField{
			Name:   "cert_error",
			System: true,
		})
		domains.Fields.Add(&core.NumberField{
			Name:   "cert_attempt",
			System: true,
		})
		domains.Fields.Add(&core.DateField{
			Name:   "cert_requested",
			System: true,
		})

		if err := app.Save(domains); err != nil {
			return err
		}

		// 2. Eliminar la colección cert_requests
		// Usamos el literal "cert_requests" porque la constante será borrada pronto.
		certReq, err := app.FindCollectionByNameOrId("cert_requests")
		if err == nil && certReq != nil {
			if err := app.Delete(certReq); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		domains, err := app.FindCollectionByNameOrId(collections.ServicesDomains)
		if err == nil {
			domains.Fields.RemoveByName("cert_status")
			domains.Fields.RemoveByName("cert_not_before")
			domains.Fields.RemoveByName("cert_not_after")
			domains.Fields.RemoveByName("cert_error")
			_ = app.Save(domains)
		}
		return nil
	})
}

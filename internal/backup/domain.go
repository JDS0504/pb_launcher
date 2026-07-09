package backup

import (
	"fmt"
	"pb_launcher/collections"
	"pb_launcher/utils/domainutil"

	"github.com/pocketbase/pocketbase/core"
)

// CreateFriendlyDomain genera y registra un subdominio friendly para un servicio de forma segura en PocketBase,
// eliminando dominios huérfanos preexistentes si es necesario.
func CreateFriendlyDomain(app core.App, serviceRecord *core.Record, domainBase string) error {
	friendlyDomain, err := domainutil.GenerateFriendlyDomain(serviceRecord.GetString("name"), domainBase)
	if err != nil {
		return nil
	}

	domainCollection, err := app.FindCachedCollectionByNameOrId(collections.ServicesDomains)
	if err != nil {
		return err
	}

	existing, err := app.FindFirstRecordByFilter(
		collections.ServicesDomains,
		"domain = {:domain}",
		map[string]any{"domain": friendlyDomain},
	)
	if err == nil && existing != nil {
		serviceId := existing.GetString("service")
		isOrphanOrDeleted := false
		if serviceId != "" {
			existingService, err := app.FindRecordById(collections.Services, serviceId)
			if err != nil || existingService == nil {
				isOrphanOrDeleted = true
			} else {
				serviceDeleted := existingService.GetDateTime("deleted")
				if !serviceDeleted.IsZero() {
					isOrphanOrDeleted = true
				}
			}
		} else {
			isOrphanOrDeleted = true
		}

		if isOrphanOrDeleted {
			_ = app.Delete(existing)
		} else {
			return fmt.Errorf("el nombre '%s' no está disponible porque el dominio '%s' ya está en uso", serviceRecord.GetString("name"), friendlyDomain)
		}
	}

	domainRecord := core.NewRecord(domainCollection)
	domainRecord.Set("domain", friendlyDomain)
	domainRecord.Set("service", []string{serviceRecord.Id})
	domainRecord.Set("use_https", "yes")
	domainRecord.Set("cert_status", "pending")

	return app.Save(domainRecord)
}

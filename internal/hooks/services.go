package hooks

import (
	"errors"
	"fmt"
	"pb_launcher/collections"
	"pb_launcher/configs"
	"pb_launcher/internal/proxy/domain"
	"regexp"
	"slices"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

func sanitizeToSlug(name string) string {
	slug := strings.ToLower(name)
	slug = slugRegex.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}

func AddServiceHooks(app *pocketbase.PocketBase,
	serviceDiscovery *domain.ServiceDiscovery,
	cnf configs.Config,
) {
	app.OnRecordCreateRequest(collections.Services).
		BindFunc(func(e *core.RecordRequestEvent) error {
			if e.Auth == nil {
				return errors.New("unauthorized: no auth record found")
			}

			name := e.Record.GetString("name")
			slug := sanitizeToSlug(name)
			if slug == "" {
				return apis.NewBadRequestError("el nombre del servicio no es válido", nil)
			}

			domainBase := cnf.GetDomain()
			parts := strings.SplitN(domainBase, ".", 2)
			rootDomain := domainBase
			if len(parts) > 1 {
				rootDomain = parts[1]
			}
			friendlyDomain := fmt.Sprintf("%s.%s", slug, rootDomain)

			existing, err := e.App.FindFirstRecordByFilter(
				collections.ServicesDomains,
				"domain = {:domain}",
				dbx.Params{"domain": friendlyDomain},
			)
			if err == nil && existing != nil {
				serviceId := existing.GetString("service")
				isOrphanOrDeleted := false
				if serviceId != "" {
					serviceRecord, err := e.App.FindRecordById(collections.Services, serviceId)
					if err != nil || serviceRecord == nil {
						isOrphanOrDeleted = true
					} else {
						serviceDeleted := serviceRecord.GetDateTime("deleted")
						if !serviceDeleted.IsZero() {
							isOrphanOrDeleted = true
						}
					}
				} else {
					isOrphanOrDeleted = true
				}

				if isOrphanOrDeleted {
					_ = e.App.Delete(existing)
				} else {
					return apis.NewBadRequestError(fmt.Sprintf("el nombre '%s' no está disponible porque el dominio '%s' ya está en uso", name, friendlyDomain), nil)
				}
			}

			restart_policy := e.Record.GetString("restart_policy")
			if !slices.Contains([]string{"no", "on-failure"}, restart_policy) {
				restart_policy = "no"
			}

			e.Record.Set("boot_completed", "no")
			e.Record.Set("restart_policy", restart_policy)

			e.Record.Set("status", "idle")

			return e.Next()
		})

	app.OnRecordUpdateRequest(collections.Services).BindFunc(func(e *core.RecordRequestEvent) error {
		updatedName := e.Record.GetString("name")
		updatedPolicy := e.Record.Get("restart_policy")
		deleted := e.Record.GetDateTime("deleted")

		currentRecord, err := e.App.FindRecordById(e.Collection, e.Record.GetString("id"))
		if err != nil {
			return err
		}

		oldName := currentRecord.GetString("name")
		if oldName != updatedName && deleted.IsZero() {
			oldSlug := sanitizeToSlug(oldName)
			newSlug := sanitizeToSlug(updatedName)
			if oldSlug != newSlug {
				domainBase := cnf.GetDomain()
				parts := strings.SplitN(domainBase, ".", 2)
				rootDomain := domainBase
				if len(parts) > 1 {
					rootDomain = parts[1]
				}
				oldFriendlyDomain := fmt.Sprintf("%s.%s", oldSlug, rootDomain)
				newFriendlyDomain := fmt.Sprintf("%s.%s", newSlug, rootDomain)

				existing, err := e.App.FindFirstRecordByFilter(
					collections.ServicesDomains,
					"domain = {:domain}",
					dbx.Params{"domain": newFriendlyDomain},
				)
				if err == nil && existing != nil {
					if existing.GetString("service") != e.Record.Id {
						serviceId := existing.GetString("service")
						isOrphanOrDeleted := false
						if serviceId != "" {
							serviceRecord, err := e.App.FindRecordById(collections.Services, serviceId)
							if err != nil || serviceRecord == nil {
								isOrphanOrDeleted = true
							} else {
								serviceDeleted := serviceRecord.GetDateTime("deleted")
								if !serviceDeleted.IsZero() {
									isOrphanOrDeleted = true
								}
							}
						} else {
							isOrphanOrDeleted = true
						}

						if isOrphanOrDeleted {
							_ = e.App.Delete(existing)
						} else {
							return apis.NewBadRequestError(fmt.Sprintf("el nombre '%s' no está disponible porque el dominio '%s' ya está en uso", updatedName, newFriendlyDomain), nil)
						}
					}
				}

				oldDomainRecord, err := e.App.FindFirstRecordByFilter(
					collections.ServicesDomains,
					"domain = {:domain} && service = {:service}",
					dbx.Params{"domain": oldFriendlyDomain, "service": e.Record.Id},
				)
				if err == nil && oldDomainRecord != nil {
					oldDomainRecord.Set("domain", newFriendlyDomain)
					if err := e.App.Save(oldDomainRecord); err != nil {
						return fmt.Errorf("failed to update domain name: %w", err)
					}
				} else {
					domainCollection, err := e.App.FindCachedCollectionByNameOrId(collections.ServicesDomains)
					if err == nil {
						domainRecord := core.NewRecord(domainCollection)
						domainRecord.Set("domain", newFriendlyDomain)
						domainRecord.Set("service", e.Record.Id)
						domainRecord.Set("use_https", "yes")
						_ = e.App.Save(domainRecord)
					}
				}
			}
		}

		currentRecord.Set("name", updatedName)
		currentRecord.Set("restart_policy", updatedPolicy)
		currentRecord.Set("deleted", deleted)

		e.Record = currentRecord
		if err := e.Next(); err != nil {
			return err
		}
		if !deleted.IsZero() {
			domains, err := e.App.FindAllRecords(
				collections.ServicesDomains,
				dbx.HashExp{"service": e.Record.Id},
			)
			if err == nil {
				for _, domainRecord := range domains {
					_ = e.App.Delete(domainRecord)
				}
			}

			comandCollection, err := e.App.FindCachedCollectionByNameOrId(collections.ServicesComands)
			if err != nil {
				return err
			}
			record := core.NewRecord(comandCollection)

			record.Set("service", e.Record.Id)
			record.Set("action", "stop")
			record.Set("status", "pending")
			record.Set("error_message", "")
			record.Set("executed", nil)

			if err := e.App.Save(record); err != nil {
				return err
			}
		}
		return nil
	})

	app.OnRecordAfterCreateSuccess(collections.Services).BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("status") == "restoring" {
			return e.Next()
		}

		name := e.Record.GetString("name")
		slug := sanitizeToSlug(name)
		domainBase := cnf.GetDomain()
		parts := strings.SplitN(domainBase, ".", 2)
		rootDomain := domainBase
		if len(parts) > 1 {
			rootDomain = parts[1]
		}
		friendlyDomain := fmt.Sprintf("%s.%s", slug, rootDomain)

		domainCollection, err := e.App.FindCachedCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}
		domainRecord := core.NewRecord(domainCollection)
		domainRecord.Set("domain", friendlyDomain)
		domainRecord.Set("service", e.Record.Id)
		domainRecord.Set("use_https", "yes")

		if err := e.App.Save(domainRecord); err != nil {
			return err
		}

		comandCollection, err := e.App.FindCachedCollectionByNameOrId(collections.ServicesComands)
		if err != nil {
			return err
		}
		record := core.NewRecord(comandCollection)

		record.Set("service", e.Record.Id)
		record.Set("action", "start")
		record.Set("status", "pending")
		record.Set("error_message", "")
		record.Set("executed", nil)

		if err := e.App.Save(record); err != nil {
			return err
		}
		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess(collections.Services).
		BindFunc(func(e *core.RecordEvent) error {
			if err := e.Next(); err != nil {
				return err
			}
			serviceDiscovery.InvalidateServiceCacheByID(e.Record.Id)
			return nil
		})

}


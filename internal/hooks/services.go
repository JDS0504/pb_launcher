package hooks

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"pb_launcher/collections"
	"pb_launcher/configs"
	launcherdomain "pb_launcher/internal/launcher/domain"
	"pb_launcher/internal/proxy/domain"
	"pb_launcher/utils/domainutil"
	"slices"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func validateUniqueFriendlyDomain(app core.App, newName string, cnf configs.Config, currentServiceId string) (string, error) {
	newFriendlyDomain, err := domainutil.GenerateFriendlyDomain(newName, cnf.GetDomain())
	if err != nil {
		return "", apis.NewBadRequestError("el nombre del servicio no es válido", nil)
	}

	existing, err := app.FindFirstRecordByFilter(
		collections.ServicesDomains,
		"domain = {:domain}",
		dbx.Params{"domain": newFriendlyDomain},
	)
	
	if err == nil && existing != nil {
		existingServiceId := existing.GetString("service")
		if existingServiceId != currentServiceId {
			isOrphan := existingServiceId == ""
			if !isOrphan {
				_, err := app.FindRecordById(collections.Services, existingServiceId)
				isOrphan = err != nil
			}
			if isOrphan {
				_ = app.Delete(existing)
			} else {
				return "", apis.NewBadRequestError(fmt.Sprintf("el nombre '%s' no está disponible porque el dominio '%s' ya está en uso", newName, newFriendlyDomain), nil)
			}
		}
	}
	return newFriendlyDomain, nil
}

func AddServiceHooks(app *pocketbase.PocketBase,
	serviceDiscovery *domain.ServiceDiscovery,
	cnf configs.Config,
	lm *launcherdomain.LauncherManager,
) {
	app.OnRecordCreateRequest(collections.Services).
		BindFunc(func(e *core.RecordRequestEvent) error {
			if e.Auth == nil {
				return errors.New("unauthorized: no auth record found")
			}

			name := e.Record.GetString("name")
			_, err := validateUniqueFriendlyDomain(e.App, name, cnf, "")
			if err != nil {
				return err
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
		updatedCpuQuota := e.Record.Get("cpu_quota")
		updatedMemoryLimit := e.Record.Get("memory_limit")

		currentRecord, err := e.App.FindRecordById(e.Collection, e.Record.GetString("id"))
		if err != nil {
			return err
		}

		oldName := currentRecord.GetString("name")
		if oldName != updatedName {
			// Detener el proceso con el nombre antiguo
			lm.StopServiceIfRunning(currentRecord.Id)

			// Renombrar la carpeta físicamente
			oldDir := filepath.Join(lm.DataDir(), oldName)
			newDir := filepath.Join(lm.DataDir(), updatedName)
			if _, err := os.Stat(oldDir); err == nil {
				if err := os.Rename(oldDir, newDir); err != nil {
					return fmt.Errorf("error renaming service directory: %w", err)
				}
			}
			oldSlug := domainutil.SanitizeToSlug(oldName)
			newSlug := domainutil.SanitizeToSlug(updatedName)
			
			if oldSlug != newSlug {
				newFriendlyDomain, err := validateUniqueFriendlyDomain(e.App, updatedName, cnf, e.Record.Id)
				if err != nil {
					return err
				}

				rootDomain := domainutil.RootDomain(cnf.GetDomain())
				domainRecords, err := e.App.FindAllRecords(
					collections.ServicesDomains,
					dbx.NewExp("service = {:service}", dbx.Params{"service": e.Record.Id}),
				)
				if err == nil {
					var autogenRecords []*core.Record
					for _, rec := range domainRecords {
						dom := rec.GetString("domain")
						if strings.HasSuffix(dom, "."+rootDomain) {
							autogenRecords = append(autogenRecords, rec)
						}
					}

					if len(autogenRecords) > 0 {
						first := autogenRecords[0]
						first.Set("domain", newFriendlyDomain)
						if err := e.App.Save(first); err != nil {
							return fmt.Errorf("failed to update domain name: %w", err)
						}
						for i := 1; i < len(autogenRecords); i++ {
							_ = e.App.Delete(autogenRecords[i])
						}
					} else {
						domainCollection, err := e.App.FindCachedCollectionByNameOrId(collections.ServicesDomains)
						if err == nil {
							domainRecord := core.NewRecord(domainCollection)
							domainRecord.Set("domain", newFriendlyDomain)
							domainRecord.Set("service", []string{e.Record.Id})
							domainRecord.Set("use_https", "yes")
							domainRecord.Set("cert_status", "pending")
							_ = e.App.Save(domainRecord)
						}
					}
				}
			}
		}

		currentRecord.Set("name", updatedName)
		currentRecord.Set("restart_policy", updatedPolicy)
		currentRecord.Set("cpu_quota", updatedCpuQuota)
		currentRecord.Set("memory_limit", updatedMemoryLimit)

		e.Record = currentRecord
		return e.Next()
	})

	app.OnRecordAfterCreateSuccess(collections.Services).BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("status") == "restoring" {
			return e.Next()
		}
		name := e.Record.GetString("name")
		friendlyDomain, err := domainutil.GenerateFriendlyDomain(name, cnf.GetDomain())
		if err != nil {
			return fmt.Errorf("invalid service name: %w", err)
		}

		domainCollection, err := e.App.FindCachedCollectionByNameOrId(collections.ServicesDomains)
		if err != nil {
			return err
		}
		domainRecord := core.NewRecord(domainCollection)
		domainRecord.Set("domain", friendlyDomain)
		domainRecord.Set("service", []string{e.Record.Id})
		domainRecord.Set("use_https", "yes")
		domainRecord.Set("cert_status", "pending")

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
			_ = serviceDiscovery.InvalidateServiceCache(e.Record.Id, e.Record.GetString("name"))
			return nil
		})

	// Detener el proceso ANTES de que PocketBase elimine el registro.
	app.OnRecordDeleteRequest(collections.Services).
		BindFunc(func(e *core.RecordRequestEvent) error {
			lm.StopServiceIfRunning(e.Record.Id)
			return e.Next()
		})

	// Borrar el directorio de datos DESPUES de la eliminación exitosa en BD.
	app.OnRecordAfterDeleteSuccess(collections.Services).
		BindFunc(func(e *core.RecordEvent) error {
			if err := e.Next(); err != nil {
				return err
			}
			name := e.Record.GetString("name")
			serviceDir := filepath.Join(lm.DataDir(), name)
			if err := os.RemoveAll(serviceDir); err != nil {
				slog.Error("failed to remove service data directory",
					"serviceName", name, "path", serviceDir, "error", err)
			} else {
				slog.Info("service data directory removed",
					"serviceName", name, "path", serviceDir)
			}
			return nil
		})
}


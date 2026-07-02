package hooks

import (
	"context"
	"log/slog"
	"pb_launcher/collections"
	"pb_launcher/configs"
	"pb_launcher/internal/certificates/tlscommon"
	certmanager "pb_launcher/internal/certmanager/domain"
	"pb_launcher/internal/certmanager/domain/repositories"
	"pb_launcher/internal/proxy/domain"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func AddServiceDomainsHooks(
	app *pocketbase.PocketBase,
	repository repositories.CertRequestRepository,
	planner *certmanager.CertRequestPlannerUsecase,
	domainDiscovery *domain.DomainServiceDiscovery,
	store tlscommon.Store,
	cnf configs.Config,
) {
	app.OnRecordsListRequest(collections.ServicesDomains).BindFunc(
		func(e *core.RecordsListRequestEvent) error {
			baseCollecion, err := e.App.FindCollectionByNameOrId(collections.ServicesDomains)
			if err != nil {
				return nil
			}
			baseCollecion.Fields.Add(&core.TextField{Name: "x_cert_request_state"})
			baseCollecion.Fields.Add(&core.BoolField{Name: "x_reached_max_attempt"})
			baseCollecion.Fields.Add(&core.TextField{Name: "x_failed_error_message"})
			baseCollecion.Fields.Add(&core.BoolField{Name: "x_has_valid_ssl_cert"})

			for idx, record := range e.Records {
				if record.GetString("use_https") != "yes" {
					continue
				}
				domain := record.GetString("domain")
				last, err := repository.LastByDomain(e.Request.Context(), domain)
				if err != nil || last == nil {
					continue
				}
				newRecord := core.NewRecord(baseCollecion)
				for _, field := range record.Collection().Fields {
					fieldName := field.GetName()
					fieldValue := record.Get(fieldName)
					newRecord.Set(fieldName, fieldValue)
				}

				newRecord.Set("x_cert_request_state", string(last.Status))
				newRecord.Set("x_reached_max_attempt", last.Attempt >= cnf.GetMaxDomainCertAttempts())
				newRecord.Set("x_failed_error_message", last.Message)

				cert, err := store.Resolve(domain)
				if err != nil || cert == nil {
					newRecord.Set("x_has_valid_ssl_cert", false)
				} else {
					newRecord.Set("x_has_valid_ssl_cert", cert.GetTTL() > 0)
				}
				e.Records[idx] = newRecord
			}
			return e.Next()
		})

	app.OnRecordCreateRequest(collections.ServicesDomains).
		BindFunc(validateService)

	app.OnRecordUpdateRequest(collections.ServicesDomains).
		BindFunc(validateService)

	app.OnRecordAfterCreateSuccess(collections.ServicesDomains).BindFunc(func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		domain := e.Record.GetString("domain")
		if e.Record.GetString("use_https") == "yes" {
			go func() {
				if err := planner.PostSSLDomainRequest(context.Background(), domain, false); err != nil {
					slog.Error("failed to post SSL domain request on create", "domain", domain, "error", err)
				}
			}()
		}
		return nil
	})

	app.OnRecordAfterUpdateSuccess(collections.ServicesDomains).
		BindFunc(func(e *core.RecordEvent) error {
			if err := e.Next(); err != nil {
				return err
			}
			domain := e.Record.GetString("domain")
			domainDiscovery.InvalidateDomain(domain)
			if e.Record.GetString("use_https") == "yes" {
				go func() {
					if err := planner.PostSSLDomainRequest(context.Background(), domain, false); err != nil {
						slog.Error("failed to post SSL domain request on update", "domain", domain, "error", err)
					}
				}()
			}
			return nil
		})

	app.OnRecordAfterDeleteSuccess(collections.ServicesDomains).
		BindFunc(func(e *core.RecordEvent) error {
			if err := e.Next(); err != nil {
				return err
			}
			domain := e.Record.GetString("domain")
			domainDiscovery.InvalidateDomain(domain)
			return repository.DeletePendingByDomain(e.Context, domain)
		})

}

func validateService(e *core.RecordRequestEvent) error {
	service := e.Record.GetString("service")
	if service == "" {
		slice := e.Record.GetStringSlice("service")
		if len(slice) > 0 {
			service = slice[0]
		}
	}
	if service == "" {
		return apis.NewBadRequestError("'service' is required", nil)
	}
	return e.Next()
}

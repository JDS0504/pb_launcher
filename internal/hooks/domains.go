package hooks

import (
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
		BindFunc(validateServiceOrProxyEntry)

	app.OnRecordUpdateRequest(collections.ServicesDomains).
		BindFunc(validateServiceOrProxyEntry)

	app.OnRecordAfterCreateSuccess(collections.ServicesDomains).BindFunc(func(e *core.RecordEvent) error {

		if err := e.Next(); err != nil {
			return err
		}
		domain := e.Record.GetString("domain")
		if e.Record.GetString("use_https") == "yes" {
			return planner.PostSSLDomainRequest(e.Context, domain, false)
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
				return planner.PostSSLDomainRequest(e.Context, domain, false)
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

	// region cert_requests
	app.OnRecordCreateRequest(collections.CertRequests).BindFunc(func(e *core.RecordRequestEvent) error {
		return planner.PostSSLDomainRequest(e.Request.Context(),
			e.Record.GetString("domain"),
			false,
		)
	})
}

func validateServiceOrProxyEntry(e *core.RecordRequestEvent) error {
	service := e.Record.GetString("service")
	proxyEntry := e.Record.GetString("proxy_entry")
	serveStatic := e.Record.GetBool("serve_static")
	switch {
	case service == "" && proxyEntry == "":
		return apis.NewBadRequestError("either 'service' or 'proxy_entry' is required", nil)
	case service != "" && proxyEntry != "":
		return apis.NewBadRequestError("only one of 'service' or 'proxy_entry' must be set", nil)
	case serveStatic && proxyEntry != "":
		return apis.NewBadRequestError("'serve_static' can only be used with a 'service', not with a 'proxy_entry'", nil)
	}
	return e.Next()
}

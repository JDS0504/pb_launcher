package repos

import (
	"context"
	"pb_launcher/collections"
	"pb_launcher/internal/certmanager/domain/models"
	"pb_launcher/internal/certmanager/domain/repositories"
	"pb_launcher/utils"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type CertRequestRepository struct {
	app *pocketbase.PocketBase
}

var _ repositories.CertRequestRepository = (*CertRequestRepository)(nil)

func NewCertRequestRepository(app *pocketbase.PocketBase) *CertRequestRepository {
	return &CertRequestRepository{app: app}
}

func (r *CertRequestRepository) DomainsWithHttpsEnabled(ctx context.Context) ([]string, error) {
	query := r.app.RecordQuery(collections.ServicesDomains).
		WithContext(ctx).
		Select("domain").
		AndWhere(dbx.NewExp("use_https = 'yes'"))

	var records []*core.Record
	if err := query.All(&records); err != nil {
		return nil, err
	}

	domains := make([]string, 0, len(records))
	for _, rec := range records {
		domains = append(domains, rec.GetString("domain"))
	}
	return domains, nil
}

func (r *CertRequestRepository) getDomainRecord(domain string) (*core.Record, error) {
	return r.app.FindFirstRecordByFilter(collections.ServicesDomains, "domain={:domain}", dbx.Params{"domain": domain})
}

func (r *CertRequestRepository) CreatePending(ctx context.Context, domain string, attempt int) error {
	record, err := r.getDomainRecord(domain)
	if err != nil {
		return err // domain not found
	}
	if attempt < 1 {
		attempt = 1
	}
	record.Set("cert_status", string(models.CertStatePending))
	record.Set("cert_not_before", time.Now().Add(time.Duration(attempt*attempt)*time.Minute))
	record.Set("cert_attempt", attempt)
	return r.app.Save(record)
}

func (r *CertRequestRepository) MarkAsApproved(ctx context.Context, id string) error {
	record, err := r.app.FindRecordById(collections.ServicesDomains, id)
	if err != nil {
		return err
	}
	record.Set("cert_status", string(models.CertStateApproved))
	record.Set("cert_error", nil)
	record.Set("cert_requested", time.Now())
	return r.app.Save(record)
}

func (r *CertRequestRepository) MarkAsFailed(ctx context.Context, id, message string) error {
	record, err := r.app.FindRecordById(collections.ServicesDomains, id)
	if err != nil {
		return err
	}
	record.Set("cert_status", string(models.CertStateFailed))
	record.Set("cert_error", message)
	record.Set("cert_requested", time.Now())
	return r.app.Save(record)
}

func (r *CertRequestRepository) Pending(ctx context.Context) ([]models.CertRequest, error) {
	query := r.app.RecordQuery(collections.ServicesDomains).
		WithContext(ctx).
		AndWhere(dbx.NewExp("cert_status='pending'"))
	var records []*core.Record
	if err := query.All(&records); err != nil {
		return nil, err
	}

	requests := make([]models.CertRequest, 0, len(records))
	for _, rec := range records {
		requests = append(requests, mapCertRequest(rec))
	}
	return requests, nil
}

func (r *CertRequestRepository) PendingByDomain(ctx context.Context, domain string) ([]models.CertRequest, error) {
	exp := dbx.NewExp(
		"domain={:domain} AND cert_status='pending'",
		dbx.Params{"domain": domain},
	)

	query := r.app.RecordQuery(collections.ServicesDomains).
		WithContext(ctx).
		AndWhere(exp).
		OrderBy("updated desc")

	var records []*core.Record
	if err := query.All(&records); err != nil {
		return nil, err
	}

	requests := make([]models.CertRequest, 0, len(records))
	for _, rec := range records {
		requests = append(requests, mapCertRequest(rec))
	}
	return requests, nil
}

func (r *CertRequestRepository) DeletePendingByDomain(ctx context.Context, domain string) error {
	const qry = "UPDATE services_domains SET cert_status='' WHERE domain = {:domain} AND cert_status = 'pending'"
	_, err := r.app.DB().NewQuery(qry).
		Bind(dbx.Params{"domain": domain}).
		WithContext(ctx).
		Execute()
	return err
}

func (r *CertRequestRepository) LastByDomain(ctx context.Context, domain string) (*models.CertRequest, error) {
	query := r.app.RecordQuery(collections.ServicesDomains).
		WithContext(ctx).
		AndWhere(dbx.NewExp("domain={:domain} AND cert_status != ''", dbx.Params{"domain": domain})).
		OrderBy("updated desc").
		Limit(1)

	var records []*core.Record
	if err := query.All(&records); err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, repositories.ErrCertRequestNotFound
	}

	req := mapCertRequest(records[0])
	return &req, nil
}

func mapCertRequest(rec *core.Record) models.CertRequest {
	return models.CertRequest{
		ID:        rec.Id, // Note: this is now the services_domains ID
		Domain:    rec.GetString("domain"),
		Status:    models.CertRequestState(rec.GetString("cert_status")),
		NotBefore: utils.Ptr(rec.GetDateTime("cert_not_before").Time()),
		Attempt:   rec.GetInt("cert_attempt"),
		Message:   utils.Ptr(rec.GetString("cert_error")),
		Created:   rec.GetDateTime("created").Time(), // Using domain creation time
		Requested: utils.Ptr(rec.GetDateTime("cert_requested").Time()),
	}
}

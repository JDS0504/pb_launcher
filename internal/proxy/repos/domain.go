package repos

import (
	"context"
	"pb_launcher/collections"
	"pb_launcher/internal/proxy/domain/repositories"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

type DomainTargetRepository struct {
	app *pocketbase.PocketBase
}

var _ repositories.DomainTargetRepository = (*DomainTargetRepository)(nil)

func NewDomainTargetRepository(app *pocketbase.PocketBase) *DomainTargetRepository {
	return &DomainTargetRepository{app: app}
}

func (r *DomainTargetRepository) FindByDomain(ctx context.Context, domain string) (*repositories.DomainTarget, error) {
	exp := dbx.NewExp("domain={:domain}", dbx.Params{"domain": domain})
	records, err := r.app.FindAllRecords(collections.ServicesDomains, exp)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, repositories.ErrNotFound
	}
	rec := records[0]
	service := rec.GetString("service")
	if service != "" {
		serveStatic := rec.GetBool("serve_static")
		return &repositories.DomainTarget{
			Service:     &service,
			ServeStatic: serveStatic,
		}, nil
	}
	proxyEntry := rec.GetString("proxy_entry")
	if proxyEntry != "" {
		return &repositories.DomainTarget{
			ProxyEntry: &proxyEntry,
		}, nil
	}
	return nil, repositories.ErrNotFound
}


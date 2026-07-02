package repos

import (
	"context"
	"fmt"
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
	errs := r.app.ExpandRecord(rec, []string{"service"}, nil)
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to expand service: %v", errs)
	}
	
	service := rec.GetString("service")
	serviceName := ""
	if expanded := rec.ExpandedOne("service"); expanded != nil {
		serviceName = expanded.GetString("name")
	}

	if service != "" {
		return &repositories.DomainTarget{
			Service:     service,
			ServiceName: serviceName,
		}, nil
	}
	return nil, repositories.ErrNotFound
}


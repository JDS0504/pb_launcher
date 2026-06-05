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
	service := records[0].GetString("service")
	if service != "" {
		return &repositories.DomainTarget{Service: &service}, nil
	}
	return nil, repositories.ErrNotFound
}

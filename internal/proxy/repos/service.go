package repos

import (
	"context"
	"database/sql"
	"errors"
	"pb_launcher/collections"
	"pb_launcher/internal/proxy/domain/dtos"
	"pb_launcher/internal/proxy/domain/repositories"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

type ServiceRepository struct {
	app *pocketbase.PocketBase
}

var _ repositories.ServiceRepository = (*ServiceRepository)(nil)

func NewServiceRepository(app *pocketbase.PocketBase) *ServiceRepository {
	return &ServiceRepository{app: app}
}

func (r *ServiceRepository) findRunningServiceByFilter(filter string, params dbx.Params) (*dtos.RunningServiceDto, error) {
	record, err := r.app.FindFirstRecordByFilter(collections.Services, filter, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "sql: no rows in result set" {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	return &dtos.RunningServiceDto{
		ID:   record.Id,
		IP:   record.GetString("ip"),
		Port: record.GetInt("port"),
	}, nil
}

func (r *ServiceRepository) FindRunningServiceByID(ctx context.Context, id string) (*dtos.RunningServiceDto, error) {
	return r.findRunningServiceByFilter("id = {:id} && (deleted = null || deleted = '') && status = 'running'", dbx.Params{"id": id})
}

func (r *ServiceRepository) FindRunningServiceByName(ctx context.Context, name string) (*dtos.RunningServiceDto, error) {
	return r.findRunningServiceByFilter("name = {:name} && (deleted = null || deleted = '') && status = 'running'", dbx.Params{"name": name})
}

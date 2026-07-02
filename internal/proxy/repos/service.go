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

func (r *ServiceRepository) FindServiceByIDOrName(ctx context.Context, idOrName string) (*dtos.ServiceDto, error) {
	record, err := r.app.FindFirstRecordByFilter(collections.Services, "id = {:idOrName} || name = {:idOrName}", dbx.Params{"idOrName": idOrName})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "sql: no rows in result set" {
			return nil, repositories.ErrNotFound
		}
		return nil, err
	}

	return &dtos.ServiceDto{
		ID:     record.Id,
		Name:   record.GetString("name"),
		Status: record.GetString("status"),
		IP:     record.GetString("ip"),
		Port:   record.GetInt("port"),
	}, nil
}

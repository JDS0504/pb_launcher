package repositories

import (
	"context"
	"errors"
	"pb_launcher/internal/proxy/domain/dtos"
)

var ErrNotFound = errors.New("not found")

type ServiceRepository interface {
	FindServiceByIDOrName(ctx context.Context, idOrName string) (*dtos.ServiceDto, error)
}

package repositories

import (
	"context"
	"errors"
	"pb_launcher/internal/proxy/domain/dtos"
)

var ErrNotFound = errors.New("not found")

type ServiceRepository interface {
	FindRunningServiceByID(ctx context.Context, id string) (*dtos.RunningServiceDto, error)
	FindRunningServiceByName(ctx context.Context, name string) (*dtos.RunningServiceDto, error)
}

package repositories

import (
	"context"
	"pb_launcher/internal/launcher/domain/models"
)

type ServiceRepository interface {
	Services(ctx context.Context) ([]models.Service, error)
	RunningServices(ctx context.Context) ([]models.Service, error)
	FindService(ctx context.Context, id string) (*models.Service, error)
	FindRelease(ctx context.Context, id string) (*models.Release, error)

	MarkServiceStoped(ctx context.Context, id string) error
	MarkServiceFailure(ctx context.Context, id string, errorMessage string) error
	MarkServiceRunning(ctx context.Context, id string, listenIplistenIp, port string) error
	UpdateServiceRelease(ctx context.Context, serviceID, releaseID string) error
	SetServiceInstallToken(ctx context.Context, serviceID string, _pb_install string) error
	CleanServiceInstallToken(ctx context.Context, _pb_install string) error
	UpdateSuperuser(ctx context.Context, serviceID, email, password string) error
}

package repositories

import (
	"context"
	"pb_launcher/internal/launcher/domain/models"
)

type CommandsRepository interface {
	PublishStartComand(ctx context.Context, serviceID string) error
	PublishStartCommand(ctx context.Context, serviceID string) error
	GetPendingCommands(ctx context.Context) ([]models.ServiceCommand, error)
	MarkCommandSuccess(ctx context.Context, id string) error
	MarkCommandError(ctx context.Context, id string, errorMessage string) error
}

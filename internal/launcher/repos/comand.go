package repos

import (
	"context"
	"pb_launcher/collections"
	"pb_launcher/internal/launcher/domain/models"
	"pb_launcher/internal/launcher/domain/repositories"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type CommandsRepository struct {
	app *pocketbase.PocketBase
}

var _ repositories.CommandsRepository = (*CommandsRepository)(nil)

func NewCommandsRepository(app *pocketbase.PocketBase) *CommandsRepository {
	return &CommandsRepository{app: app}
}

func (c *CommandsRepository) PublishStartComand(ctx context.Context, serviceID string) error {
	return c.PublishStartCommand(ctx, serviceID)
}

func (c *CommandsRepository) PublishStartCommand(ctx context.Context, serviceID string) error {
	comandCollection, err := c.app.FindCachedCollectionByNameOrId(collections.ServicesComands)
	if err != nil {
		return err
	}
	record := core.NewRecord(comandCollection)

	record.Set("service", serviceID)
	record.Set("action", "start")
	record.Set("status", "pending")
	record.Set("error_message", "")
	record.Set("executed", nil)

	return c.app.Save(record)
}

// GetPendingCommands implements repositories.CommandsRepository.
func (c *CommandsRepository) GetPendingCommands(ctx context.Context) ([]models.ServiceCommand, error) {
	var records []*core.Record
	query := c.app.RecordQuery(collections.ServicesComands).
		Select("id", "service", "action", "target_release").
		AndWhere(dbx.NewExp("status = 'pending'")).
		OrderBy("created")
	if err := query.All(&records); err != nil {
		return nil, err
	}
	comands := make([]models.ServiceCommand, 0, len(records))
	for _, r := range records {
		var action models.CommandAction
		switch r.GetString("action") {
		case "stop":
			action = models.ActionStop
		case "start":
			action = models.ActionStart
		case "restart":
			action = models.ActionRestart
		case "upgrade":
			action = models.ActionUpgrade
		}
		comands = append(comands, models.ServiceCommand{
			ID:            r.Id,
			Service:       r.GetString("service"),
			Action:        action,
			TargetRelease: r.GetString("target_release"),
		})
	}
	return comands, nil
}

// MarkCommandError implements repositories.CommandsRepository.
func (c *CommandsRepository) MarkCommandError(ctx context.Context, id string, errorMessage string) error {
	record, err := c.app.FindRecordById(collections.ServicesComands, id)
	if err != nil {
		return err
	}
	record.Set("executed", time.Now())
	record.Set("error_message", errorMessage)
	record.Set("status", "error")
	return c.app.Save(record)
}

// MarkCommandSuccess implements repositories.CommandsRepository.
func (c *CommandsRepository) MarkCommandSuccess(ctx context.Context, id string) error {
	record, err := c.app.FindRecordById(collections.ServicesComands, id)
	if err != nil {
		return err
	}
	record.Set("executed", time.Now())
	record.Set("error_message", nil)
	record.Set("status", "success")
	return c.app.Save(record)
}

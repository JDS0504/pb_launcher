package migrations

import (
	"pb_launcher/collections"
	"pb_launcher/utils"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}

		logs := core.NewBaseCollection(collections.OperationLogs)
		logs.Fields.Add(
			&core.RelationField{
				Name:         "service",
				CollectionId: services.Id,
				System:       true,
				MinSelect:    0,
				MaxSelect:    1,
			},
			&core.TextField{
				Name:     "operation",
				System:   true,
				Required: true,
			},
			&core.SelectField{
				Name:      "status",
				System:    true,
				Required:  true,
				MaxSelect: 1,
				Values:    []string{"success", "error"},
			},
			&core.TextField{
				Name:   "message",
				System: true,
			},
			&core.JSONField{
				Name:   "metadata",
				System: true,
			},
			&core.AutodateField{
				Name:     "created",
				System:   true,
				OnCreate: true,
			},
		)
		logs.Indexes = append(logs.Indexes,
			`CREATE INDEX idx_operation_logs_service ON operation_logs(service)`,
			`CREATE INDEX idx_operation_logs_created ON operation_logs(created)`,
			`CREATE INDEX idx_operation_logs_operation ON operation_logs(operation)`,
		)
		logs.ListRule = utils.StrPointer(`@request.auth.id != ""`)
		logs.ViewRule = utils.StrPointer(`@request.auth.id != ""`)
		return app.Save(logs)
	}, func(app core.App) error {
		logs, err := app.FindCollectionByNameOrId(collections.OperationLogs)
		if err != nil {
			return err
		}
		return app.Delete(logs)
	})
}

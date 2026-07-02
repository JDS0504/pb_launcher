package migrations

import (
	"log/slog"
	"pb_launcher/collections"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		records, err := app.FindAllRecords(
			collections.ServicesDomains,
			dbx.NewExp("domain LIKE '%apps.%'"),
		)
		if err != nil {
			slog.Info("no legacy apps domains found or collection missing", "error", err)
			return nil
		}

		for _, record := range records {
			domainName := record.GetString("domain")
			if err := app.Delete(record); err != nil {
				slog.Error("failed to delete legacy apps domain record", "domain", domainName, "error", err)
			} else {
				slog.Info("deleted legacy apps domain record successfully", "domain", domainName)
			}
		}

		return nil
	}, func(app core.App) error {
		return nil
	})
}

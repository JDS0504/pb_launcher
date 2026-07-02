package migrations

import (
	"log/slog"
	"pb_launcher/collections"
	"pb_launcher/configs"
	"pb_launcher/utils/domainutil"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		cnf, err := configs.LoadConfigs("config.yml")
		if err != nil {
			slog.Error("failed to load configs in migration", "error", err)
			return nil
		}
		rootDomain := domainutil.RootDomain(cnf.GetDomain())
		suffix := "." + rootDomain
		appsSuffix := ".apps." + rootDomain

		records, err := app.FindAllRecords(collections.ServicesDomains)
		if err != nil {
			slog.Info("no domains found or collection missing", "error", err)
			return nil
		}

		for _, record := range records {
			domainName := record.GetString("domain")
			// Delete if it is a platform subdomain (ends with .sistemasimpulsa.com or .apps.sistemasimpulsa.com)
			if strings.HasSuffix(domainName, suffix) || strings.HasSuffix(domainName, appsSuffix) {
				if err := app.Delete(record); err != nil {
					slog.Error("failed to delete legacy system domain record", "domain", domainName, "error", err)
				} else {
					slog.Info("deleted legacy system domain record successfully", "domain", domainName)
				}
			}
		}

		return nil
	}, func(app core.App) error {
		return nil
	})
}

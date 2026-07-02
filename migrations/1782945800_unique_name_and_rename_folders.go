package migrations

import (
	"log/slog"
	"os"
	"path/filepath"
	"pb_launcher/collections"
	"pb_launcher/configs"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err != nil {
			return err
		}

		// 1. Modificar esquema para hacer 'name' Unique
		services.Indexes = append(services.Indexes, "CREATE UNIQUE INDEX `idx_services_name` ON `services` (`name`)")

		if err := app.Save(services); err != nil {
			return err
		}

		// 2. Renombrar carpetas de datos de <id> a <name>
		dataDir := "./data"
		if cnf, err := configs.LoadConfigs("config.yml"); err == nil && cnf != nil {
			dataDir = cnf.GetDataDir()
		}

		records, err := app.FindAllRecords(collections.Services)
		if err != nil {
			return err
		}

		for _, record := range records {
			id := record.Id
			name := record.GetString("name")
			
			if name == "" {
				continue
			}

			oldPath := filepath.Join(dataDir, id)
			newPath := filepath.Join(dataDir, name)

			// Solo intentamos renombrar si la carpeta antigua existe y la nueva NO existe
			if _, err := os.Stat(oldPath); err == nil {
				if _, err := os.Stat(newPath); os.IsNotExist(err) {
					if err := os.Rename(oldPath, newPath); err != nil {
						slog.Error("error renaming service directory", "id", id, "name", name, "error", err)
					} else {
						slog.Info("migrated service directory", "old", oldPath, "new", newPath)
					}
				}
			}
		}

		return nil
	}, func(app core.App) error {
		services, err := app.FindCollectionByNameOrId(collections.Services)
		if err == nil {
			// Remover el index
			for i, idx := range services.Indexes {
				if idx == "CREATE UNIQUE INDEX `idx_services_name` ON `services` (`name`)" {
					services.Indexes = append(services.Indexes[:i], services.Indexes[i+1:]...)
					break
				}
			}
			_ = app.Save(services)
		}
		
		// No revertimos el renombrado de carpetas porque el sistema nuevo las espera con el nombre
		return nil
	})
}

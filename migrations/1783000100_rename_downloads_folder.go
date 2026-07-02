package migrations

import (
	"log/slog"
	"os"
	"path/filepath"
	"pb_launcher/configs"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		downloadDir := "./downloads"
		if cnf, err := configs.LoadConfigs("config.yml"); err == nil && cnf != nil {
			downloadDir = cnf.GetDownloadDir()
		}

		oldPath := filepath.Join(downloadDir, "pb91u2l315h29a5")
		newPath := filepath.Join(downloadDir, "pocketbase")

		if _, err := os.Stat(oldPath); err == nil {
			if _, err := os.Stat(newPath); os.IsNotExist(err) {
				if err := os.Rename(oldPath, newPath); err != nil {
					slog.Error("failed to rename legacy downloads directory", "old", oldPath, "new", newPath, "error", err)
				} else {
					slog.Info("successfully migrated legacy downloads directory to new structure", "old", oldPath, "new", newPath)
				}
			}
		}

		return nil
	}, func(app core.App) error {
		return nil
	})
}

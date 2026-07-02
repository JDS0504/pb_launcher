package migrations

import (
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/core"
)

// Migración que limpia servicios en soft-delete previos usando SQL directo.
// El campo `deleted` permanece en el schema (System:true, no se puede eliminar
// via app.Save). El frontend ya no lo usa: ahora hace DELETE real.
// El campo queda en BD pero nunca se escribe ni se lee desde el código.
func init() {
	m.Register(func(app core.App) error {
		db := app.DB()

		// Limpiar hijos antes que el padre (FK Required:true)
		_, _ = db.NewQuery(`
			DELETE FROM services_domains WHERE service IN (
				SELECT id FROM services WHERE deleted IS NOT NULL AND deleted != ''
			)
		`).Execute()
		_, _ = db.NewQuery(`
			DELETE FROM comands WHERE service IN (
				SELECT id FROM services WHERE deleted IS NOT NULL AND deleted != ''
			)
		`).Execute()
		_, _ = db.NewQuery(`
			DELETE FROM operation_logs WHERE service IN (
				SELECT id FROM services WHERE deleted IS NOT NULL AND deleted != ''
			)
		`).Execute()
		_, _ = db.NewQuery(`
			DELETE FROM services WHERE deleted IS NOT NULL AND deleted != ''
		`).Execute()

		return nil
	}, func(app core.App) error {
		return nil // sin rollback necesario (datos ya eliminados)
	})
}

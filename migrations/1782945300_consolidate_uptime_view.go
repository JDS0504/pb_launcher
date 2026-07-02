package migrations

import (
	"pb_launcher/utils"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Migración SSOT: elimina la vista SQL intermedia `vista_tabla_service_uptime`
// y consolida toda la lógica directamente en el ViewQuery de la colección
// PocketBase `service_uptime_view`.
//
// Antes (2 fuentes de verdad):
//   vista_tabla_service_uptime  ← vista SQLite con el cálculo completo
//          ↓  SELECT * FROM
//   service_uptime_view         ← colección PocketBase (wrapper vacío)
//
// Después (1 fuente de verdad):
//   service_uptime_view         ← colección PocketBase con el SQL completo
//                                  directamente en ViewQuery
//
// También elimina WHERE deleted = '' ya que el campo `deleted` fue removido
// en la migración 1782945200_cascade_and_hard_delete.go.
func init() {
	const fullSQL = `
		WITH service_epochs AS (
		  SELECT
		    id AS service_id,
		    name AS service_name,
		    status AS service_status,
		    CAST(strftime('%s', created) AS INTEGER) AS created_epoch,
		    CAST(strftime('%s', 'now') AS INTEGER) AS now_epoch
		  FROM services
		),
		service_limits AS (
		  SELECT
		    service_id,
		    service_name,
		    service_status,
		    created_epoch,
		    now_epoch,
		    CASE
		      WHEN (now_epoch - 24 * 3600) < created_epoch THEN created_epoch
		      ELSE now_epoch - 24 * 3600
		    END AS start_24h,
		    CASE
		      WHEN (now_epoch - 7 * 24 * 3600) < created_epoch THEN created_epoch
		      ELSE now_epoch - 7 * 24 * 3600
		    END AS start_7d
		  FROM service_epochs
		),
		successful_logs AS (
		  SELECT
		    service,
		    operation,
		    CAST(strftime('%s', created) AS INTEGER) AS created_epoch
		  FROM operation_logs
		  WHERE status = 'success'
		),
		ordered_intervals AS (
		  SELECT
		    l.service,
		    l.created_epoch,
		    CASE WHEN l.operation IN ('start', 'wakeup', 'restart') THEN 1 ELSE 0 END AS is_active,
		    COALESCE(
		      LEAD(l.created_epoch) OVER (PARTITION BY l.service ORDER BY l.created_epoch ASC),
		      CAST(strftime('%s', 'now') AS INTEGER)
		    ) AS next_epoch
		  FROM successful_logs l
		),
		active_intervals AS (
		  SELECT service, created_epoch, next_epoch
		  FROM ordered_intervals
		  WHERE is_active = 1
		),
		stats_24h AS (
		  SELECT
		    sl.service_id,
		    COALESCE(
		      SUM(
		        CASE
		          WHEN ai.next_epoch > sl.start_24h
		          THEN MAX(0, MIN(ai.next_epoch, sl.now_epoch) - MAX(ai.created_epoch, sl.start_24h))
		          ELSE 0
		        END
		      ), 0
		    ) AS active_sec
		  FROM service_limits sl
		  LEFT JOIN active_intervals ai ON ai.service = sl.service_id
		  GROUP BY sl.service_id
		),
		stats_7d AS (
		  SELECT
		    sl.service_id,
		    COALESCE(
		      SUM(
		        CASE
		          WHEN ai.next_epoch > sl.start_7d
		          THEN MAX(0, MIN(ai.next_epoch, sl.now_epoch) - MAX(ai.created_epoch, sl.start_7d))
		          ELSE 0
		        END
		      ), 0
		    ) AS active_sec
		  FROM service_limits sl
		  LEFT JOIN active_intervals ai ON ai.service = sl.service_id
		  GROUP BY sl.service_id
		)
		SELECT
		  sl.service_id AS id,
		  sl.service_name,
		  sl.service_status,
		  ROUND(MIN(100.0, (s24.active_sec * 100.0) / MAX(1, sl.now_epoch - sl.start_24h)), 2) AS uptime_24h,
		  ROUND(s24.active_sec / 3600.0, 1) AS active_hours_24h,
		  ROUND(((sl.now_epoch - sl.start_24h) - s24.active_sec) / 3600.0, 1) AS inactive_hours_24h,
		  ROUND(MIN(100.0, (s7.active_sec * 100.0) / MAX(1, sl.now_epoch - sl.start_7d)), 2) AS uptime_7d,
		  ROUND(s7.active_sec / 3600.0, 1) AS active_hours_7d,
		  ROUND(((sl.now_epoch - sl.start_7d) - s7.active_sec) / 3600.0, 1) AS inactive_hours_7d
		FROM service_limits sl
		JOIN stats_24h s24 ON s24.service_id = sl.service_id
		JOIN stats_7d s7 ON s7.service_id = sl.service_id
	`

	m.Register(func(app core.App) error {
		// 1. Eliminar la vista SQL intermedia — el SQL vive ahora en ViewQuery
		if _, err := app.DB().NewQuery(
			`DROP VIEW IF EXISTS vista_tabla_service_uptime`,
		).Execute(); err != nil {
			return err
		}

		// 2. Actualizar la colección PocketBase con el SQL completo directamente
		viewCol, err := app.FindCollectionByNameOrId("service_uptime_view")
		if err != nil {
			return err
		}
		viewCol.ViewQuery = fullSQL
		viewCol.ListRule = utils.StrPointer(`@request.auth.id != ""`)
		viewCol.ViewRule = utils.StrPointer(`@request.auth.id != ""`)
		return app.Save(viewCol)
	}, func(app core.App) error {
		// Downgrade: restaurar vista SQL intermedia y volver al SELECT * FROM
		restoreView := `
			CREATE VIEW IF NOT EXISTS vista_tabla_service_uptime AS
			` + fullSQL
		if _, err := app.DB().NewQuery(restoreView).Execute(); err != nil {
			return err
		}

		viewCol, err := app.FindCollectionByNameOrId("service_uptime_view")
		if err != nil {
			return err
		}
		viewCol.ViewQuery = `
			SELECT
				id, service_name, service_status,
				uptime_24h, active_hours_24h, inactive_hours_24h,
				uptime_7d, active_hours_7d, inactive_hours_7d
			FROM vista_tabla_service_uptime
		`
		viewCol.ListRule = utils.StrPointer(`@request.auth.id != ""`)
		viewCol.ViewRule = utils.StrPointer(`@request.auth.id != ""`)
		return app.Save(viewCol)
	})
}

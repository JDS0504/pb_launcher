package systemstatus

import (
	"context"
	"pb_launcher/utils/processstats"

	"go.uber.org/fx"
)

// Module is the Fx injection point for the system status monitoring module.
// Arranca el DefaultMonitor de processstats junto con el ciclo de vida de la aplicación.
var Module = fx.Module("systemstatus",
	fx.Invoke(RegisterRoutes),
	fx.Invoke(func(lc fx.Lifecycle) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				processstats.DefaultMonitor.Start(context.Background())
				return nil
			},
		})
	}),
)

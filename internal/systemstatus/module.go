package systemstatus

import "go.uber.org/fx"

// Module is the Fx injection point for the system status monitoring module
var Module = fx.Module("systemstatus",
	fx.Invoke(RegisterRoutes),
)

package backup

import "go.uber.org/fx"

var Module = fx.Module("backup",
	fx.Provide(NewManager),
	fx.Invoke(RegisterRoutes),
)

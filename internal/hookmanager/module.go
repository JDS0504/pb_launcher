package hookmanager

import "go.uber.org/fx"

var Module = fx.Module("hookmanager",
	fx.Provide(NewManager),
	fx.Invoke(RegisterRoutes),
)

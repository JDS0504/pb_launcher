package repositorymanager

import "go.uber.org/fx"

var Module = fx.Module("repositorymanager",
	fx.Provide(NewManager),
	fx.Invoke(RegisterRoutes),
)

package filemanager

import "go.uber.org/fx"

var Module = fx.Module("filemanager",
	fx.Provide(NewManager),
	fx.Invoke(RegisterRoutes),
)

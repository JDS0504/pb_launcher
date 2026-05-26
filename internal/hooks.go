package internal

import (
	"pb_launcher/internal/hooks"

	"go.uber.org/fx"
)

var Module = fx.Module("hooks",
	fx.Invoke(hooks.RegisterAdminExistsRoute),
	fx.Invoke(hooks.RegisterServiceLogsRoute),
	fx.Invoke(hooks.RegisterUpsertServiceSuperuserRoute),
	fx.Invoke(hooks.RegisterServiceCliRoute),
	fx.Invoke(hooks.AddServiceHooks),
	fx.Invoke(hooks.AddProxyEntriesHooks),
	fx.Invoke(hooks.AddServiceDomainsHooks),
	fx.Invoke(hooks.AddComandHooks),
	fx.Invoke(hooks.AddRepositoryHooks),
)

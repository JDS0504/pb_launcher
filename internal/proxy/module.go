package proxy

import (
	"pb_launcher/internal/proxy/domain"
	"pb_launcher/internal/proxy/domain/repositories"
	"pb_launcher/internal/proxy/repos"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"proxy",
	fx.Provide(
		fx.Annotate(
			repos.NewServiceRepository,
			fx.As(new(repositories.ServiceRepository)),
		),
		fx.Annotate(
			repos.NewDomainTargetRepository,
			fx.As(new(repositories.DomainTargetRepository)),
		),
	),
	fx.Provide(
		domain.NewServiceDiscovery,
		domain.NewDomainServiceDiscovery,
	),
	fx.Provide(NewDynamicReverseProxyDiscovery),
	fx.Provide(NewDynamicReverseProxy),
	fx.Invoke(RunHttpProxy, RunHTTPSProxy, PrintProxyInfo),
)

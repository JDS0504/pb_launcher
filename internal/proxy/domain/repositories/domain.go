package repositories

import "context"

type DomainTarget struct {
	Service     *string
	ProxyEntry  *string
	ServeStatic bool // cuando true, sirve pb_public desde disco sin encender PocketBase
}

type DomainTargetRepository interface {
	FindByDomain(ctx context.Context, domain string) (*DomainTarget, error)
}

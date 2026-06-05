package repositories

import "context"

type DomainTarget struct {
	Service *string
}

type DomainTargetRepository interface {
	FindByDomain(ctx context.Context, domain string) (*DomainTarget, error)
}

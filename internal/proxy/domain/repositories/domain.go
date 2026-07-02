package repositories

import "context"

type DomainTarget struct {
	Service     string // id del servicio asociado al dominio
	ServiceName string // nombre del servicio asociado al dominio
}

type DomainTargetRepository interface {
	FindByDomain(ctx context.Context, domain string) (*DomainTarget, error)
}

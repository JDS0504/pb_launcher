package domain

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"log/slog"
	"pb_launcher/internal/proxy/domain/dtos"
	"pb_launcher/internal/proxy/domain/repositories"

	"time"

	"github.com/allegro/bigcache/v3"
)

type ServiceDiscovery struct {
	repo  repositories.ServiceRepository
	cache *bigcache.BigCache
}

func init() {
	gob.Register(&dtos.ServiceDto{})
}

func NewServiceDiscovery(repo repositories.ServiceRepository) (*ServiceDiscovery, error) {

	cache, err := bigcache.New(context.Background(), bigcache.Config{
		Shards:           256,
		LifeWindow:       15 * time.Second, // debe ser << autosleep_idle_timeout para no reiniciar el timer de inactividad
		CleanWindow:      2 * time.Minute,
		MaxEntrySize:     512,
		Verbose:          false,
		HardMaxCacheSize: 128,
		StatsEnabled:     false,
	})
	if err != nil {
		return nil, err
	}

	return &ServiceDiscovery{
		repo:  repo,
		cache: cache,
	}, nil
}

func (s *ServiceDiscovery) FindServiceByIDOrName(ctx context.Context, idOrName string) (*dtos.ServiceDto, error) {
	if data, err := s.cache.Get(idOrName); err == nil {
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		var dto dtos.ServiceDto
		if err := dec.Decode(&dto); err == nil {
			return &dto, nil
		}
	} else if !errors.Is(err, bigcache.ErrEntryNotFound) {
		slog.Warn("failed to access cache", "idOrName", idOrName, "error", err)
	}

	dto, err := s.repo.FindServiceByIDOrName(ctx, idOrName)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(dto); err == nil {
		if err := s.cache.Set(idOrName, buf.Bytes()); err != nil {
			slog.Warn("failed to cache service", "idOrName", idOrName, "error", err)
		}
	}

	return dto, nil
}

func (s *ServiceDiscovery) InvalidateServiceCache(id, name string) error {
	_ = s.cache.Delete(id)
	_ = s.cache.Delete(name)
	slog.Info("invalidated service cache", "service_id", id, "service_name", name)
	return nil
}

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
	gob.Register(&dtos.RunningServiceDto{})
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

func (s *ServiceDiscovery) findCachedRunningService(ctx context.Context, key string, fetchFunc func() (*dtos.RunningServiceDto, error)) (*dtos.RunningServiceDto, error) {
	if data, err := s.cache.Get(key); err == nil {
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		var dto dtos.RunningServiceDto
		if err := dec.Decode(&dto); err == nil {
			return &dto, nil
		}
	} else if !errors.Is(err, bigcache.ErrEntryNotFound) {
		slog.Warn("failed to access cache", "key", key, "error", err)
	}

	dto, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(dto); err == nil {
		if err := s.cache.Set(key, buf.Bytes()); err != nil {
			slog.Warn("failed to cache service", "key", key, "error", err)
		}
	}

	return dto, nil
}

func (s *ServiceDiscovery) FindRunningServiceByID(ctx context.Context, id string) (*dtos.RunningServiceDto, error) {
	return s.findCachedRunningService(ctx, id, func() (*dtos.RunningServiceDto, error) {
		return s.repo.FindRunningServiceByID(ctx, id)
	})
}

func (s *ServiceDiscovery) FindRunningServiceByName(ctx context.Context, name string) (*dtos.RunningServiceDto, error) {
	return s.findCachedRunningService(ctx, name, func() (*dtos.RunningServiceDto, error) {
		return s.repo.FindRunningServiceByName(ctx, name)
	})
}

func (s *ServiceDiscovery) InvalidateServiceCacheByID(id string) error {
	err := s.cache.Delete(id)
	if err != nil && !errors.Is(err, bigcache.ErrEntryNotFound) {
		slog.Error("failed to invalidate cache", "id", id, "error", err)
		return err
	}
	slog.Info("invalidated service cache", "service_id", id)
	return nil
}

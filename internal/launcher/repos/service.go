package repos

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"pb_launcher/collections"
	"pb_launcher/internal/launcher/domain/models"
	"pb_launcher/internal/launcher/domain/repositories"
	"regexp"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type ServiceRepository struct {
	app *pocketbase.PocketBase
}

var _ repositories.ServiceRepository = (*ServiceRepository)(nil)

func NewServiceRepository(app *pocketbase.PocketBase) *ServiceRepository {
	return &ServiceRepository{app: app}
}

func (s *ServiceRepository) services(ids ...string) ([]models.Service, error) {
	qry := `
		select 
			s.id, 
			s.name,
			s.release,
			s.status, 
			s.restart_policy, 
			r.version, 
			s._pb_install,
			s.boot_user_email,
			s.boot_user_password,
			s.ip,
			s.port,
			s.cpu_quota,
			s.memory_limit
		from services s
		inner join releases r on s."release" = r.id`

	var quoted []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		quoted = append(quoted, fmt.Sprintf("'%s'", id))
	}
	if len(quoted) > 0 {
		quotedStr := strings.Join(quoted, ",")
		qry += fmt.Sprintf(" and (s.id in (%s) OR s.name in (%s))", quotedStr, quotedStr)
	}
	db := s.app.DB()

	results := []dbx.NullStringMap{}
	if err := db.NewQuery(qry).All(&results); err != nil {
		log.Fatal(err)
	}

	services := make([]models.Service, 0, len(results))
	for _, row := range results {
		id, _ := row["id"]
		name, _ := row["name"]
		release, _ := row["release"]
		status, _ := row["status"]
		restartPolicy, _ := row["restart_policy"]
		version, _ := row["version"]
		_pb_install, _ := row["_pb_install"]
		bootUserEmail, _ := row["boot_user_email"]
		bootUserPassword, _ := row["boot_user_password"]
		ip, _ := row["ip"]
		port, _ := row["port"]
		cpuQuota, _ := row["cpu_quota"]
		memoryLimit, _ := row["memory_limit"]

		ExecFilePattern, err := regexp.Compile(`pocketbase.*`)
		if err != nil {
			slog.Warn("invalid exec file pattern", "error", err)
			continue
		}

		services = append(services, models.Service{
			ID:                id.String,
			Name:              name.String,
			ReleaseID:         release.String,
			Status:            models.ServiceStatus(status.String),
			RestartPolicy:     models.RestartPolicy(restartPolicy.String),
			Version:           version.String,
			ExecFilePattern:   ExecFilePattern,
			BootPBInstallPath: _pb_install.String,
			BootUserEmail:     bootUserEmail.String,
			BootUserPassword:  bootUserPassword.String,
			IP:                ip.String,
			Port:              port.String,
			CpuQuota:          cpuQuota.String,
			MemoryLimit:       memoryLimit.String,
		})
	}

	return services, nil
}

// Services implements repositories.ServiceRepository.
func (s *ServiceRepository) Services(ctx context.Context) ([]models.Service, error) {
	return s.services()
}

// RunningServices implements repositories.ServiceRepository.
func (s *ServiceRepository) RunningServices(ctx context.Context) ([]models.Service, error) {
	services, err := s.Services(ctx)
	if err != nil {
		return nil, err
	}
	results := []models.Service{}
	for _, service := range services {
		if service.Status == models.Running {
			results = append(results, service)
		}
	}
	return results, nil
}

// FindService implements repositories.ServiceRepository.
func (s *ServiceRepository) FindService(ctx context.Context, id string) (*models.Service, error) {
	services, err := s.services(id)
	if err != nil {
		return nil, err
	}
	if len(services) == 0 {
		return nil, fmt.Errorf("service not found: %s", id)
	}
	return &services[0], nil
}

func (s *ServiceRepository) FindRelease(ctx context.Context, id string) (*models.Release, error) {
	record, err := s.app.FindRecordById(collections.Releases, id)
	if err != nil {
		return nil, err
	}
	return &models.Release{
		ID:           record.Id,
		Version:      record.GetString("version"),
	}, nil
}

// updateRecord is a helper to find a service record, apply changes, and save it, ensuring DRY.
func (s *ServiceRepository) updateRecord(id string, updateFn func(r *core.Record)) error {
	record, err := s.app.FindRecordById(collections.Services, id)
	if err != nil {
		return err
	}
	updateFn(record)
	return s.app.Save(record)
}

// MarkServiceStoped implements repositories.ServiceRepository.
func (s *ServiceRepository) MarkServiceStoped(ctx context.Context, id string) error {
	return s.updateRecord(id, func(record *core.Record) {
		record.Set("status", string(models.Stopped))
		record.Set("error_message", nil)
	})
}

// MarkServiceSleeping implements repositories.ServiceRepository.
func (s *ServiceRepository) MarkServiceSleeping(ctx context.Context, id string) error {
	return s.updateRecord(id, func(record *core.Record) {
		record.Set("status", string(models.Sleeping))
		record.Set("error_message", nil)
	})
}

// MarkServiceFailure implements repositories.ServiceRepository.
func (s *ServiceRepository) MarkServiceFailure(ctx context.Context, id string, errorMessage string) error {
	return s.updateRecord(id, func(record *core.Record) {
		record.Set("status", string(models.Failure))
		record.Set("error_message", errorMessage)
	})
}

// MarkServiceRunning implements repositories.ServiceRepository.
func (s *ServiceRepository) MarkServiceRunning(ctx context.Context, id, listenIp, port string) error {
	return s.updateRecord(id, func(record *core.Record) {
		record.Set("status", string(models.Running))
		record.Set("last_started", time.Now())
		record.Set("error_message", nil)
		record.Set("ip", listenIp)
		record.Set("port", port)
	})
}

// UpdateServiceRelease implements repositories.ServiceRepository.
func (s *ServiceRepository) UpdateServiceRelease(ctx context.Context, serviceID, releaseID string) error {
	return s.updateRecord(serviceID, func(record *core.Record) {
		record.Set("release", releaseID)
		record.Set("error_message", nil)
	})
}

// SetServiceInstallToken implements repositories.ServiceRepository.
func (s *ServiceRepository) SetServiceInstallToken(ctx context.Context, id string, _pb_install string) error {
	return s.updateRecord(id, func(record *core.Record) {
		record.Set("_pb_install", _pb_install)
	})
}

// CleanPbInstallToken implements repositories.ServiceRepository.
func (s *ServiceRepository) CleanServiceInstallToken(ctx context.Context, _pb_install string) error {
	db := s.app.DB()

	qry := fmt.Sprintf(
		"UPDATE %s SET _pb_install = '' WHERE _pb_install = {:token}",
		collections.Services,
	)

	_, execErr := db.NewQuery(qry).
		WithContext(ctx).
		Bind(dbx.Params{"token": _pb_install}).
		Execute()

	if execErr != nil {
		slog.Error("update services table", "error", execErr)
	}
	return nil
}

func (s *ServiceRepository) UpdateSuperuser(ctx context.Context, serviceID, email, password string) error {
	db := s.app.DB()

	query := fmt.Sprintf(
		`UPDATE %s 
			SET boot_user_email = {:email},
				boot_user_password = {:password},
				_pb_install = ''
			WHERE id = {:id}`,
		collections.Services,
	)
	_, execErr := db.NewQuery(query).
		WithContext(ctx).
		Bind(dbx.Params{"id": serviceID, "email": email, "password": password}).
		Execute()

	return execErr
}

// ClearCurrentSnapshot implements repositories.ServiceRepository.
// Limpia el current_snapshot_id del servicio, indicando que el estado en disco
// ya no corresponde a ningún snapshot registrado (fue modificado).
func (s *ServiceRepository) ClearCurrentSnapshot(ctx context.Context, serviceID string) error {
	return s.updateRecord(serviceID, func(record *core.Record) {
		record.Set("current_snapshot_id", "")
	})
}


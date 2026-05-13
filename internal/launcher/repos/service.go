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
			r.repository, 
			rpo.exec_file_pattern,
			s._pb_install,
			s.boot_user_email,
			s.boot_user_password,
			s.deleted
		from services s
		inner join releases r on s."release" = r.id
		inner join repositories rpo on rpo.id = r.repository`

	var quoted []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		quoted = append(quoted, fmt.Sprintf("'%s'", id))
	}
	if len(quoted) > 0 {
		qry += fmt.Sprintf(" and s.id in (%s)", strings.Join(quoted, ","))
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
		repository, _ := row["repository"]
		execPattern, _ := row["exec_file_pattern"]
		_pb_install, _ := row["_pb_install"]
		bootUserEmail, _ := row["boot_user_email"]
		bootUserPassword, _ := row["boot_user_password"]
		deleted, _ := row["deleted"]

		ExecFilePattern, err := regexp.Compile(execPattern.String)
		if err != nil {
			slog.Warn("invalid exec file pattern", "error", err, "pattern", execPattern)
			continue
		}

		services = append(services, models.Service{
			ID:                id.String,
			Name:              name.String,
			ReleaseID:         release.String,
			Status:            models.ServiceStatus(status.String),
			RestartPolicy:     models.RestartPolicy(restartPolicy.String),
			Version:           version.String,
			RepositoryID:      repository.String,
			ExecFilePattern:   ExecFilePattern,
			BootPBInstallPath: _pb_install.String,
			BootUserEmail:     bootUserEmail.String,
			BootUserPassword:  bootUserPassword.String,
			Deleted:           deleted.String,
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
		RepositoryID: record.GetString("repository"),
		Version:      record.GetString("version"),
	}, nil
}

// MarkServiceStoped implements repositories.ServiceRepository.
func (s *ServiceRepository) MarkServiceStoped(ctx context.Context, id string) error {

	record, err := s.app.FindRecordById(collections.Services, id)
	if err != nil {
		return err
	}

	record.Set("status", string(models.Stopped))
	record.Set("error_message", nil)

	if err := s.app.Save(record); err != nil {
		return err
	}

	return nil
}

// MarkServiceFailure implements repositories.ServiceRepository.
func (s *ServiceRepository) MarkServiceFailure(ctx context.Context, id string, errorMessage string) error {

	record, err := s.app.FindRecordById(collections.Services, id)
	if err != nil {
		return err
	}

	record.Set("status", string(models.Failure))
	record.Set("error_message", errorMessage)

	if err := s.app.Save(record); err != nil {
		return err
	}

	return nil
}

// MarkServiceRunning implements repositories.ServiceRepository.
func (s *ServiceRepository) MarkServiceRunning(ctx context.Context, id, listenIp, port string) error {

	record, err := s.app.FindRecordById(collections.Services, id)
	if err != nil {
		return err
	}

	record.Set("status", string(models.Running))
	record.Set("last_started", time.Now())
	record.Set("error_message", nil)
	record.Set("ip", listenIp)
	record.Set("port", port)

	if err := s.app.Save(record); err != nil {
		return err
	}

	return nil
}

func (s *ServiceRepository) UpdateServiceRelease(ctx context.Context, serviceID, releaseID string) error {
	record, err := s.app.FindRecordById(collections.Services, serviceID)
	if err != nil {
		return err
	}

	record.Set("release", releaseID)
	record.Set("error_message", nil)

	return s.app.Save(record)
}

// SetPbInstallToken implements repositories.ServiceRepository.
func (s *ServiceRepository) SetServiceInstallToken(ctx context.Context, id string, _pb_install string) error {
	record, err := s.app.FindRecordById(collections.Services, id)
	if err != nil {
		return err
	}
	record.Set("_pb_install", _pb_install)
	if err := s.app.Save(record); err != nil {
		return err
	}
	return nil
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

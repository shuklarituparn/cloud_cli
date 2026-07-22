package intrumentation

import (
	"context"
	"log/slog"
	"time"

	application "github.com/shuklarituparn/cloud_cli/internal/application/profile"
	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"
	"github.com/shuklarituparn/cloud_cli/internal/observability/metrics"
)

type Recorder interface {
	Record(context.Context, metrics.Observation) error
}

type Service struct {
	next     application.UseCases
	recorder Recorder
	logger   *slog.Logger
	now      func() time.Time
}

func NewService(next application.UseCases, recorder Recorder, logger *slog.Logger) *Service {
	return &Service{next: next, recorder: recorder, logger: logger, now: time.Now}
}

func (s *Service) Create(ctx context.Context, command application.CreateCommand) (p domain.Profile, err error) {
	start := s.now()
	defer func() { s.record(ctx, "profile.create", start, err, nil) }()
	return s.next.Create(ctx, command)
}

func (s *Service) Get(ctx context.Context, name string) (p domain.Profile, err error) {
	start := s.now()
	defer func() { s.record(ctx, "profile.get", start, err, nil) }()
	return s.next.Get(ctx, name)
}

func (s *Service) List(ctx context.Context) (profiles []domain.Profile, err error) {
	start := s.now()
	defer func() {
		var count *int
		if err == nil {
			count = new(len(profiles))
		}
		s.record(ctx, "profile.list", start, err, count)
	}()
	return s.next.List(ctx)
}

func (s *Service) Delete(ctx context.Context, name string) (err error) {
	start := s.now()
	defer func() { s.record(ctx, "profile.delete", start, err, nil) }()
	return s.next.Delete(ctx, name)
}

func (s *Service) record(ctx context.Context, operation string, start time.Time, operationErr error, profilesTotal *int) {
	if err := s.recorder.Record(ctx, metrics.Observation{Operation: operation, Err: operationErr, Duration: s.now().Sub(start), ProfilesTotal: profilesTotal}); err != nil {
		s.logger.WarnContext(ctx, "metrics recording failed", "operation", operation, "error", err)
	}
}

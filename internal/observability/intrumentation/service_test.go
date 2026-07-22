package intrumentation

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	application "github.com/shuklarituparn/cloud_cli/internal/application/profile"
	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"

	"github.com/shuklarituparn/cloud_cli/internal/observability/metrics"
)

type useCasesStub struct {
	createFn func(context.Context, application.CreateCommand) (domain.Profile, error)
	getFn    func(context.Context, string) (domain.Profile, error)
	listFn   func(context.Context) ([]domain.Profile, error)
	deleteFn func(context.Context, string) error
}

func (s *useCasesStub) Create(ctx context.Context, c application.CreateCommand) (domain.Profile, error) {
	return s.createFn(ctx, c)
}
func (s *useCasesStub) Get(ctx context.Context, n string) (domain.Profile, error) {
	return s.getFn(ctx, n)
}
func (s *useCasesStub) List(ctx context.Context) ([]domain.Profile, error) { return s.listFn(ctx) }
func (s *useCasesStub) Delete(ctx context.Context, n string) error         { return s.deleteFn(ctx, n) }

type recorderStub struct {
	observations []metrics.Observation
	err          error
}

func (r *recorderStub) Record(_ context.Context, observation metrics.Observation) error {
	r.observations = append(r.observations, observation)
	return r.err
}

func TestServiceRecordsAllOperations(t *testing.T) {
	p, _ := domain.New("test", "alice", "platform")
	next := &useCasesStub{
		createFn: func(context.Context, application.CreateCommand) (domain.Profile, error) { return p, nil },
		getFn:    func(context.Context, string) (domain.Profile, error) { return domain.Profile{}, domain.ErrNotFound },
		listFn:   func(context.Context) ([]domain.Profile, error) { return []domain.Profile{p}, nil },
		deleteFn: func(context.Context, string) error { return nil },
	}
	recorder := &recorderStub{}
	service := NewService(next, recorder, slog.New(slog.NewTextHandler(io.Discard, nil)))
	now := time.Unix(0, 0)
	service.now = func() time.Time { now = now.Add(time.Millisecond); return now }

	if _, err := service.Create(context.Background(), application.CreateCommand{}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), "test"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := service.List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}

	if len(recorder.observations) != 4 {
		t.Fatalf("observations = %d", len(recorder.observations))
	}
	want := []string{"profile.create", "profile.get", "profile.list", "profile.delete"}
	for i, operation := range want {
		if recorder.observations[i].Operation != operation {
			t.Fatalf("observation[%d] = %q", i, recorder.observations[i].Operation)
		}
		if recorder.observations[i].Duration <= 0 {
			t.Fatalf("duration[%d] = %s", i, recorder.observations[i].Duration)
		}
	}
	if recorder.observations[1].Err == nil {
		t.Fatal("failed get recorded as success")
	}
	if recorder.observations[2].ProfilesTotal == nil || *recorder.observations[2].ProfilesTotal != 1 {
		t.Fatalf("list count = %+v", recorder.observations[2].ProfilesTotal)
	}
}

func TestServiceIgnoresMetricsFailure(t *testing.T) {
	p, _ := domain.New("test", "alice", "platform")
	next := &useCasesStub{
		createFn: func(context.Context, application.CreateCommand) (domain.Profile, error) { return p, nil },
		getFn:    func(context.Context, string) (domain.Profile, error) { return p, nil },
		listFn:   func(context.Context) ([]domain.Profile, error) { return nil, nil },
		deleteFn: func(context.Context, string) error { return nil },
	}
	service := NewService(next, &recorderStub{err: errors.New("disk full")}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := service.Create(context.Background(), application.CreateCommand{}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

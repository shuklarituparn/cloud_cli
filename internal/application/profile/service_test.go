package profile

import (
	"context"
	"errors"
	"testing"

	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"
)

type repositoryStub struct {
	createFn func(context.Context, domain.Profile) error
	getFn    func(context.Context, domain.Name) (domain.Profile, error)
	listFn   func(context.Context) ([]domain.Profile, error)
	deleteFn func(context.Context, domain.Name) error
}

func newRepositoryStub() *repositoryStub {
	return &repositoryStub{
		createFn: func(context.Context, domain.Profile) error { return nil },
		getFn:    func(context.Context, domain.Name) (domain.Profile, error) { return domain.Profile{}, nil },
		listFn:   func(context.Context) ([]domain.Profile, error) { return nil, nil },
		deleteFn: func(context.Context, domain.Name) error { return nil },
	}
}

func (r *repositoryStub) Create(ctx context.Context, p domain.Profile) error {
	return r.createFn(ctx, p)
}
func (r *repositoryStub) Get(ctx context.Context, n domain.Name) (domain.Profile, error) {
	return r.getFn(ctx, n)
}
func (r *repositoryStub) List(ctx context.Context) ([]domain.Profile, error) { return r.listFn(ctx) }
func (r *repositoryStub) Delete(ctx context.Context, n domain.Name) error    { return r.deleteFn(ctx, n) }

func TestServiceCreate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		repository := newRepositoryStub()
		var persisted domain.Profile
		repository.createFn = func(_ context.Context, p domain.Profile) error { persisted = p; return nil }
		service := NewService(repository)
		got, err := service.Create(context.Background(), CreateCommand{Name: "test", User: "alice", Project: "platform"})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if got != persisted || got.Name.String() != "test" {
			t.Fatalf("Create() = %+v, persisted = %+v", got, persisted)
		}
	})

	t.Run("validation happens before persistence", func(t *testing.T) {
		repository := newRepositoryStub()
		called := false
		repository.createFn = func(context.Context, domain.Profile) error { called = true; return nil }
		service := NewService(repository)
		_, err := service.Create(context.Background(), CreateCommand{Name: "../bad", User: "u", Project: "p"})
		if !errors.Is(err, domain.ErrInvalidName) {
			t.Fatalf("Create() error = %v", err)
		}
		if called {
			t.Fatal("repository called for invalid command")
		}
	})

	t.Run("repository error is wrapped", func(t *testing.T) {
		repository := newRepositoryStub()
		repository.createFn = func(context.Context, domain.Profile) error { return domain.ErrAlreadyExists }
		_, err := NewService(repository).Create(context.Background(), CreateCommand{Name: "test", User: "u", Project: "p"})
		if !errors.Is(err, domain.ErrAlreadyExists) {
			t.Fatalf("Create() error = %v", err)
		}
	})
}

func TestServiceGetListDelete(t *testing.T) {
	t.Parallel()
	p, _ := domain.New("test", "alice", "platform")

	t.Run("get success", func(t *testing.T) {
		repository := newRepositoryStub()
		repository.getFn = func(_ context.Context, name domain.Name) (domain.Profile, error) {
			if name != p.Name {
				t.Fatalf("name = %q", name)
			}
			return p, nil
		}
		got, err := NewService(repository).Get(context.Background(), "test")
		if err != nil || got != p {
			t.Fatalf("Get() = %+v, %v", got, err)
		}
	})

	t.Run("get validates name", func(t *testing.T) {
		_, err := NewService(newRepositoryStub()).Get(context.Background(), "../bad")
		if !errors.Is(err, domain.ErrInvalidName) {
			t.Fatalf("Get() error = %v", err)
		}
	})

	t.Run("get wraps repository error", func(t *testing.T) {
		repository := newRepositoryStub()
		repository.getFn = func(context.Context, domain.Name) (domain.Profile, error) {
			return domain.Profile{}, domain.ErrNotFound
		}
		_, err := NewService(repository).Get(context.Background(), "test")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("Get() error = %v", err)
		}
	})

	t.Run("list success", func(t *testing.T) {
		repository := newRepositoryStub()
		repository.listFn = func(context.Context) ([]domain.Profile, error) { return []domain.Profile{p}, nil }
		got, err := NewService(repository).List(context.Background())
		if err != nil || len(got) != 1 || got[0] != p {
			t.Fatalf("List() = %+v, %v", got, err)
		}
	})

	t.Run("list wraps repository error", func(t *testing.T) {
		repository := newRepositoryStub()
		repository.listFn = func(context.Context) ([]domain.Profile, error) { return nil, context.Canceled }
		_, err := NewService(repository).List(context.Background())
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("List() error = %v", err)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		repository := newRepositoryStub()
		called := false
		repository.deleteFn = func(_ context.Context, name domain.Name) error { called = name == p.Name; return nil }
		err := NewService(repository).Delete(context.Background(), "test")
		if err != nil || !called {
			t.Fatalf("Delete() error = %v, called = %v", err, called)
		}
	})

	t.Run("delete validates name", func(t *testing.T) {
		err := NewService(newRepositoryStub()).Delete(context.Background(), "../bad")
		if !errors.Is(err, domain.ErrInvalidName) {
			t.Fatalf("Delete() error = %v", err)
		}
	})

	t.Run("delete wraps repository error", func(t *testing.T) {
		repository := newRepositoryStub()
		repository.deleteFn = func(context.Context, domain.Name) error { return domain.ErrNotFound }
		err := NewService(repository).Delete(context.Background(), "test")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("Delete() error = %v", err)
		}
	})
}

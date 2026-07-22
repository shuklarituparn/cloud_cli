package profile

import (
	"context"

	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"
)

type Repository interface {
	Create(context.Context, domain.Profile) error
	Get(context.Context, domain.Name) (domain.Profile, error)
	List(context.Context) ([]domain.Profile, error)
	Delete(context.Context, domain.Name) error
}

type UseCases interface {
	Create(context.Context, CreateCommand) (domain.Profile, error)
	Get(context.Context, string) (domain.Profile, error)
	List(context.Context) ([]domain.Profile, error)
	Delete(context.Context, string) error
}

type CreateCommand struct {
	Name    string
	User    string
	Project string
}

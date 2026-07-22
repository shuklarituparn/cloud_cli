package profile

import (
	"context"
	"fmt"

	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, command CreateCommand) (domain.Profile, error) {
	p, err := domain.New(command.Name, command.User, command.Project)
	if err != nil {
		return domain.Profile{}, err
	}
	if err := s.repository.Create(ctx, p); err != nil {
		return domain.Profile{}, fmt.Errorf("create profile: %w", err)
	}
	return p, nil
}

func (s *Service) Get(ctx context.Context, name string) (domain.Profile, error) {
	parsed, err := domain.ParseName(name)
	if err != nil {
		return domain.Profile{}, err
	}
	p, err := s.repository.Get(ctx, parsed)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("get profile: %w", err)
	}
	return p, nil
}

func (s *Service) List(ctx context.Context) ([]domain.Profile, error) {
	profiles, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}
	return profiles, nil
}

func (s *Service) Delete(ctx context.Context, name string) error {
	parsed, err := domain.ParseName(name)
	if err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, parsed); err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	return nil
}

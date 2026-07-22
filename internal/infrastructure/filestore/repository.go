package filestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"
)

type Repository struct {
	dir   string
	codec Codec
}

func NewRepository(dir string, codec Codec) *Repository {
	return &Repository{dir: dir, codec: codec}
}

func (r *Repository) Create(ctx context.Context, p domain.Profile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := domain.New(p.Name.String(), p.User, p.Project); err != nil {
		return err
	}
	path := r.path(p.Name)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %q", domain.ErrAlreadyExists, p.Name)
		}
		return fmt.Errorf("create profile file: %w", err)
	}

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(path)
		}
	}()
	if err := r.codec.Encode(file, p); err != nil {
		_ = file.Close()
		return fmt.Errorf("encode profile: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync profile file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close profile file: %w", err)
	}
	cleanup = false
	return nil
}

func (r *Repository) Get(ctx context.Context, name domain.Name) (domain.Profile, error) {
	if err := ctx.Err(); err != nil {
		return domain.Profile{}, err
	}
	path := r.path(name)
	if err := ensureRegularFile(path); err != nil {
		return domain.Profile{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return domain.Profile{}, fmt.Errorf("%w: %q", domain.ErrNotFound, name)
		}
		return domain.Profile{}, fmt.Errorf("open profile file: %w", err)
	}
	defer file.Close()
	p, err := r.codec.Decode(file, name)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	return p, nil
}

func (r *Repository) List(ctx context.Context) ([]domain.Profile, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, fmt.Errorf("read profiles directory: %w", err)
	}
	profiles := make([]domain.Profile, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || strings.ToLower(filepath.Ext(entry.Name())) != ".yaml" {
			continue
		}
		name, err := domain.ParseName(strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())))
		if err != nil {
			continue
		}
		p, err := r.Get(ctx, name)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name.String() < profiles[j].Name.String() })
	return profiles, nil
}

func (r *Repository) Delete(ctx context.Context, name domain.Name) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path := r.path(name)
	if err := ensureRegularFile(path); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %q", domain.ErrNotFound, name)
		}
		return fmt.Errorf("remove profile file: %w", err)
	}
	return nil
}

func (r *Repository) path(name domain.Name) string {
	return filepath.Join(r.dir, name.String()+".yaml")
}

func ensureRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %q", domain.ErrNotFound, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
		}
		return fmt.Errorf("inspect profile file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s is not a regular file", domain.ErrUnsafeFile, filepath.Base(path))
	}
	return nil
}

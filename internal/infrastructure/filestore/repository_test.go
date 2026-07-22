package filestore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"
)

func TestRepositoryCRUD(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	repository := NewRepository(dir, YAMLCodec{})
	p, err := domain.New("test", "alice", "platform")
	if err != nil {
		t.Fatal(err)
	}

	if err := repository.Create(context.Background(), p); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "test.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}

	got, err := repository.Get(context.Background(), p.Name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != p {
		t.Fatalf("Get() = %+v, want %+v", got, p)
	}

	profiles, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0] != p {
		t.Fatalf("List() = %+v", profiles)
	}

	if err := repository.Delete(context.Background(), p.Name); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	_, err = repository.Get(context.Background(), p.Name)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get() after delete error = %v", err)
	}
}

func TestRepositoryCreateDoesNotOverwrite(t *testing.T) {
	t.Parallel()
	repository := NewRepository(t.TempDir(), YAMLCodec{})
	p, _ := domain.New("test", "alice", "platform")
	if err := repository.Create(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), p); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("second Create() error = %v, want ErrAlreadyExists", err)
	}
}

func TestRepositoryRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "test.yaml")); err != nil {
		t.Fatal(err)
	}
	repository := NewRepository(dir, YAMLCodec{})
	name, _ := domain.ParseName("test")
	_, err := repository.Get(context.Background(), name)
	if !errors.Is(err, domain.ErrUnsafeFile) {
		t.Fatalf("Get() error = %v, want ErrUnsafeFile", err)
	}
}

package profile

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStoreCRUD(t *testing.T) {
	t.Parallel()

	store := NewStore(t.TempDir())
	first := Profile{Name: "dev", User: "alice", Project: "payments"}
	second := Profile{Name: "prod", User: "bob", Project: "core"}

	if err := store.Create(second); err != nil {
		t.Fatalf("Create(second): %v", err)
	}
	if err := store.Create(first); err != nil {
		t.Fatalf("Create(first): %v", err)
	}

	got, err := store.Get("dev")
	if err != nil {
		t.Fatalf("Get(dev): %v", err)
	}
	if !reflect.DeepEqual(got, first) {
		t.Fatalf("Get(dev) = %#v, want %#v", got, first)
	}

	profiles, err := store.List()
	if err != nil {
		t.Fatalf("List(): %v", err)
	}
	want := []Profile{first, second}
	if !reflect.DeepEqual(profiles, want) {
		t.Fatalf("List() = %#v, want %#v", profiles, want)
	}

	if err := store.Delete("dev"); err != nil {
		t.Fatalf("Delete(dev): %v", err)
	}
	if _, err := store.Get("dev"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(deleted) error = %v, want ErrNotFound", err)
	}
}

func TestCreateDoesNotOverwrite(t *testing.T) {
	t.Parallel()

	store := NewStore(t.TempDir())
	p := Profile{Name: "dev", User: "alice", Project: "payments"}

	if err := store.Create(p); err != nil {
		t.Fatalf("first Create(): %v", err)
	}
	if err := store.Create(p); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("second Create() error = %v, want ErrAlreadyExists", err)
	}
}

func TestProfileFileFormat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := NewStore(dir)
	p := Profile{Name: "test", User: "example", Project: "new-project"}

	if err := store.Create(p); err != nil {
		t.Fatalf("Create(): %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "test.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(): %v", err)
	}
	want := "user: \"example\"\nproject: \"new-project\"\n"
	if string(data) != want {
		t.Fatalf("file content = %q, want %q", data, want)
	}
}

func TestValidateName(t *testing.T) {
	t.Parallel()

	valid := []string{"test", "prod-eu", "team_2", "версия1", "v1.2"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q): %v", name, err)
		}
	}

	invalid := []string{"", ".hidden", "../escape", "a/b", "with space"}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) succeeded, want error", name)
		}
	}
}

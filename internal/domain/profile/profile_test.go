package profile

import (
	"errors"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()
	p, err := New("team-1", "alice", "billing")
	if err != nil {
		t.Fatalf("New() err = %v", err)
	}
	if p.Name.String() != "team-1" || p.User != "alice" || p.Project != "billing" {
		t.Fatalf("unexpected profile: %+v", p)
	}
}

func TestNewRejectsEmptyFields(t *testing.T) {
	t.Parallel()
	_, err := New("test", " ", "project")
	if !errors.Is(err, ErrInvalidUser) {
		t.Fatalf("error = %v", err)
	}
	_, err = New("test", "user", " ")
	if !errors.Is(err, ErrInvalidProject) {
		t.Fatalf("error = %v", err)
	}
}

func TestParseName(t *testing.T) {
	t.Parallel()

	valid := []string{"team-1", "team_1", "tea" +
		"m.1", "Команда1"}
	for _, value := range valid {
		name, err := ParseName(value)
		if err != nil || name.String() != value {
			t.Fatalf("ParseName(%q) = %q, %v", value, name, err)
		}
	}

	invalid := []string{"", "../secret", ".hidden", "name/other", "name with space", strings.Repeat("a", 129), string([]byte{0xff})}
	for _, value := range invalid {
		_, err := ParseName(value)
		if !errors.Is(err, ErrInvalidName) {
			t.Fatalf("ParseName(%q) error = %v, want ErrInvalidName", value, err)
		}
	}
}

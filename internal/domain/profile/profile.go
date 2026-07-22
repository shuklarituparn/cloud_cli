package profile

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrAlreadyExists  = errors.New("profile already exists")
	ErrNotFound       = errors.New("profile not found")
	ErrInvalidName    = errors.New("invalid profile name")
	ErrInvalidUser    = errors.New("invalid user")
	ErrInvalidProject = errors.New("invalid project")
	ErrUnsafeFile     = errors.New("unsafe profile file")
)

type Name string
type Profile struct {
	Name    Name
	User    string
	Project string
}

func ParseName(value string) (Name, error) {
	if value == "" {
		return "", fmt.Errorf("%w: must not be empty", ErrInvalidName)
	}
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("%w: must be valid UTF-8", ErrInvalidName)
	}
	if len([]rune(value)) > 128 {
		return "", fmt.Errorf("%w: must not exceed 128 characters", ErrInvalidName)
	}
	if value == "." || value == ".." || strings.HasPrefix(value, ".") {
		return "", fmt.Errorf("%w: must not start with a dot", ErrInvalidName)
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("%w: unsupported character %q", ErrInvalidName, r)
	}
	return Name(value), nil
}

func (n Name) String() string { return string(n) }

func New(name, user, project string) (Profile, error) {
	parsedName, err := ParseName(name)
	if err != nil {
		return Profile{}, err
	}
	if strings.TrimSpace(user) == "" {
		return Profile{}, fmt.Errorf("%w: must not be empty", ErrInvalidUser)
	}
	if strings.TrimSpace(project) == "" {
		return Profile{}, fmt.Errorf("%w: must not be empty", ErrInvalidProject)
	}
	return Profile{Name: parsedName, User: user, Project: project}, nil
}

package profile

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	ErrAlreadyExists = errors.New("profile already exists")
	ErrNotFound      = errors.New("profile not found")
)

type Profile struct {
	Name    string
	User    string
	Project string
}

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) Create(p Profile) error {
	if err := validateProfile(p); err != nil {
		return err
	}

	path, err := s.profilePath(p.Name)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("%w: %q", ErrAlreadyExists, p.Name)
		}
		return fmt.Errorf("create profile file: %w", err)
	}

	writeErr := encodeYAML(file, p)
	closeErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(path)
		return fmt.Errorf("write profile file: %w", writeErr)
	}
	if closeErr != nil {
		_ = os.Remove(path)
		return fmt.Errorf("close profile file: %w", closeErr)
	}

	return nil
}

func (s *Store) Get(name string) (Profile, error) {
	path, err := s.profilePath(name)
	if err != nil {
		return Profile{}, err
	}

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Profile{}, fmt.Errorf("%w: %q", ErrNotFound, name)
		}
		return Profile{}, fmt.Errorf("open profile file: %w", err)
	}
	defer file.Close()

	p, err := decodeYAML(file)
	if err != nil {
		return Profile{}, fmt.Errorf("decode %s: %w", filepath.Base(path), err)
	}
	p.Name = name
	return p, nil
}

func (s *Store) List() ([]Profile, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read profiles directory: %w", err)
	}

	profiles := make([]Profile, 0)
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".yaml" {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if err := ValidateName(name); err != nil {
			continue
		}

		p, err := s.Get(name)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})
	return profiles, nil
}

func (s *Store) Delete(name string) error {
	path, err := s.profilePath(name)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %q", ErrNotFound, name)
		}
		return fmt.Errorf("delete profile file: %w", err)
	}
	return nil
}

func (s *Store) profilePath(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	return filepath.Join(s.dir, name+".yaml"), nil
}

func validateProfile(p Profile) error {
	if err := ValidateName(p.Name); err != nil {
		return err
	}
	if strings.TrimSpace(p.User) == "" {
		return errors.New("user must not be empty")
	}
	if strings.TrimSpace(p.Project) == "" {
		return errors.New("project must not be empty")
	}
	return nil
}

func ValidateName(name string) error {
	if name == "" {
		return errors.New("profile name must not be empty")
	}
	if len([]rune(name)) > 128 {
		return errors.New("profile name must not exceed 128 characters")
	}
	if name == "." || name == ".." || strings.HasPrefix(name, ".") {
		return errors.New("profile name must not start with a dot")
	}

	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			continue
		}
		return fmt.Errorf("profile name contains unsupported character %q", r)
	}
	return nil
}

func encodeYAML(w io.Writer, p Profile) error {
	_, err := fmt.Fprintf(w, "user: %s\nproject: %s\n", strconv.Quote(p.User), strconv.Quote(p.Project))
	return err
}

func decodeYAML(r io.Reader) (Profile, error) {
	scanner := bufio.NewScanner(r)
	values := make(map[string]string, 2)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, rawValue, found := strings.Cut(line, ":")
		if !found {
			return Profile{}, fmt.Errorf("invalid line %q", line)
		}
		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)

		if key != "user" && key != "project" {
			return Profile{}, fmt.Errorf("unknown field %q", key)
		}
		if _, exists := values[key]; exists {
			return Profile{}, fmt.Errorf("duplicate field %q", key)
		}
		if rawValue == "" {
			return Profile{}, fmt.Errorf("field %q must not be empty", key)
		}

		value := rawValue
		if strings.HasPrefix(rawValue, "\"") {
			unquoted, err := strconv.Unquote(rawValue)
			if err != nil {
				return Profile{}, fmt.Errorf("invalid quoted value for %q: %w", key, err)
			}
			value = unquoted
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return Profile{}, fmt.Errorf("read YAML: %w", err)
	}

	user, hasUser := values["user"]
	project, hasProject := values["project"]
	if !hasUser || !hasProject {
		return Profile{}, errors.New("YAML must contain user and project fields")
	}
	if strings.TrimSpace(user) == "" || strings.TrimSpace(project) == "" {
		return Profile{}, errors.New("user and project must not be empty")
	}

	return Profile{User: user, Project: project}, nil
}

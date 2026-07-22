package filestore

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"
)

type Codec interface {
	Encode(io.Writer, domain.Profile) error
	Decode(io.Reader, domain.Name) (domain.Profile, error)
}

type YAMLCodec struct{}

func (YAMLCodec) Encode(w io.Writer, p domain.Profile) error {
	_, err := fmt.Fprintf(w, "user: %s\nproject: %s\n", strconv.Quote(p.User), strconv.Quote(p.Project))
	return err
}

func (YAMLCodec) Decode(r io.Reader, name domain.Name) (domain.Profile, error) {
	scanner := bufio.NewScanner(r)
	values := make(map[string]string, 2)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, rawValue, found := strings.Cut(line, ":")
		if !found {
			return domain.Profile{}, fmt.Errorf("invalid YAML line %q", line)
		}
		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)
		if key != "user" && key != "project" {
			return domain.Profile{}, fmt.Errorf("unknown YAML field %q", key)
		}
		if _, exists := values[key]; exists {
			return domain.Profile{}, fmt.Errorf("duplicate YAML field %q", key)
		}
		if rawValue == "" {
			return domain.Profile{}, fmt.Errorf("YAML field %q must not be empty", key)
		}
		value := rawValue
		if strings.HasPrefix(rawValue, "\"") {
			unquoted, err := strconv.Unquote(rawValue)
			if err != nil {
				return domain.Profile{}, fmt.Errorf("decode YAML field %q: %w", key, err)
			}
			value = unquoted
		}
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return domain.Profile{}, fmt.Errorf("read YAML: %w", err)
	}

	user, hasUser := values["user"]
	project, hasProject := values["project"]
	if !hasUser || !hasProject {
		return domain.Profile{}, errors.New("YAML must contain user and project fields")
	}
	return domain.New(name.String(), user, project)
}

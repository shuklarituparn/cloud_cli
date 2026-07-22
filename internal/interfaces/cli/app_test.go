package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	application "github.com/shuklarituparn/cloud_cli/internal/application/profile"
	"github.com/shuklarituparn/cloud_cli/internal/buildinfo"
	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"
)

type useCasesStub struct {
	createFn func(context.Context, application.CreateCommand) (domain.Profile, error)
	getFn    func(context.Context, string) (domain.Profile, error)
	listFn   func(context.Context) ([]domain.Profile, error)
	deleteFn func(context.Context, string) error
}

func newUseCasesStub() *useCasesStub {
	p, _ := domain.New("test", "alice", "platform")
	return &useCasesStub{
		createFn: func(context.Context, application.CreateCommand) (domain.Profile, error) { return p, nil },
		getFn:    func(context.Context, string) (domain.Profile, error) { return p, nil },
		listFn:   func(context.Context) ([]domain.Profile, error) { return []domain.Profile{p}, nil },
		deleteFn: func(context.Context, string) error { return nil },
	}
}
func (s *useCasesStub) Create(ctx context.Context, command application.CreateCommand) (domain.Profile, error) {
	return s.createFn(ctx, command)
}
func (s *useCasesStub) Get(ctx context.Context, name string) (domain.Profile, error) {
	return s.getFn(ctx, name)
}
func (s *useCasesStub) List(ctx context.Context) ([]domain.Profile, error) { return s.listFn(ctx) }
func (s *useCasesStub) Delete(ctx context.Context, name string) error      { return s.deleteFn(ctx, name) }

type exporterStub struct {
	prometheusFn func(context.Context, io.Writer) error
	jsonFn       func(context.Context, io.Writer) error
}

func newExporterStub() *exporterStub {
	return &exporterStub{
		prometheusFn: func(_ context.Context, w io.Writer) error { _, err := io.WriteString(w, "metric 1\n"); return err },
		jsonFn:       func(_ context.Context, w io.Writer) error { _, err := io.WriteString(w, "{}\n"); return err },
	}
}
func (e *exporterStub) WritePrometheus(ctx context.Context, w io.Writer) error {
	return e.prometheusFn(ctx, w)
}
func (e *exporterStub) WriteJSON(ctx context.Context, w io.Writer) error { return e.jsonFn(ctx, w) }

func runApp(t *testing.T, profiles application.UseCases, exporter MetricsExporter, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	app := New(profiles, exporter, buildinfo.Info{Version: "v1", Commit: "abc", Date: "today"}, &out, &errOut)
	code := app.Run(context.Background(), args)
	return code, out.String(), errOut.String()
}

func TestAppRoutingAndHelp(t *testing.T) {
	t.Parallel()
	profiles, exporter := newUseCasesStub(), newExporterStub()
	cases := []struct {
		name     string
		args     []string
		code     int
		contains string
		stderr   bool
	}{
		{"no args", nil, 2, "Usage:", true},
		{"help", []string{"help"}, 0, "profile create", false},
		{"short help", []string{"-h"}, 0, "profile create", false},
		{"unknown", []string{"wat"}, 2, `unknown command "wat"`, true},
		{"profile missing", []string{"profile"}, 2, "profile subcommand is required", true},
		{"profile unknown", []string{"profile", "wat"}, 2, `unknown profile subcommand "wat"`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errOut := runApp(t, profiles, exporter, tc.args...)
			if code != tc.code {
				t.Fatalf("code = %d", code)
			}
			text := out
			if tc.stderr {
				text = errOut
			}
			if !strings.Contains(text, tc.contains) {
				t.Fatalf("output = %q, want %q", text, tc.contains)
			}
		})
	}
}

func TestAppProfileCreate(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		profiles := newUseCasesStub()
		profiles.createFn = func(_ context.Context, command application.CreateCommand) (domain.Profile, error) {
			if command.Name != "test" || command.User != "alice" || command.Project != "platform" {
				t.Fatalf("command = %+v", command)
			}
			return domain.New(command.Name, command.User, command.Project)
		}
		code, out, errOut := runApp(t, profiles, newExporterStub(), "profile", "create", "--name=test", "--user=alice", "--project=platform")
		if code != 0 || errOut != "" || !strings.Contains(out, `profile "test" created`) {
			t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
		}
	})

	t.Run("domain error", func(t *testing.T) {
		profiles := newUseCasesStub()
		profiles.createFn = func(context.Context, application.CreateCommand) (domain.Profile, error) {
			return domain.Profile{}, domain.ErrInvalidUser
		}
		code, _, errOut := runApp(t, profiles, newExporterStub(), "profile", "create")
		if code != 2 || !strings.Contains(errOut, "create profile") {
			t.Fatalf("code=%d err=%q", code, errOut)
		}
	})

	t.Run("already exists", func(t *testing.T) {
		profiles := newUseCasesStub()
		profiles.createFn = func(context.Context, application.CreateCommand) (domain.Profile, error) {
			return domain.Profile{}, domain.ErrAlreadyExists
		}
		code, _, _ := runApp(t, profiles, newExporterStub(), "profile", "create")
		if code != 1 {
			t.Fatalf("code = %d", code)
		}
	})

	t.Run("bad flags", func(t *testing.T) {
		code, _, _ := runApp(t, newUseCasesStub(), newExporterStub(), "profile", "create", "--bad")
		if code != 2 {
			t.Fatalf("code = %d", code)
		}
	})

	t.Run("extra args", func(t *testing.T) {
		code, _, _ := runApp(t, newUseCasesStub(), newExporterStub(), "profile", "create", "extra")
		if code != 2 {
			t.Fatalf("code = %d", code)
		}
	})
}

func TestAppProfileGet(t *testing.T) {
	t.Parallel()
	profiles := newUseCasesStub()
	for _, tc := range []struct{ name, output, contains string }{
		{"yaml", "yaml", "name: test"},
		{"json", "json", `"project": "platform"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, out, errOut := runApp(t, profiles, newExporterStub(), "profile", "get", "--name=test", "--output="+tc.output)
			if code != 0 || errOut != "" || !strings.Contains(out, tc.contains) {
				t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
			}
		})
	}
	code, _, errOut := runApp(t, profiles, newExporterStub(), "profile", "get", "--name=test", "--output=xml")
	if code != 2 || !strings.Contains(errOut, "unsupported output") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}

	profiles.getFn = func(context.Context, string) (domain.Profile, error) { return domain.Profile{}, domain.ErrNotFound }
	code, _, _ = runApp(t, profiles, newExporterStub(), "profile", "get", "--name=missing")
	if code != 1 {
		t.Fatalf("not found code = %d", code)
	}
}

func TestAppProfileList(t *testing.T) {
	t.Parallel()
	profiles := newUseCasesStub()
	code, out, errOut := runApp(t, profiles, newExporterStub(), "profile", "list")
	if code != 0 || errOut != "" || !strings.Contains(out, "NAME") || !strings.Contains(out, "platform") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errOut)
	}

	code, out, _ = runApp(t, profiles, newExporterStub(), "profile", "list", "--output=json")
	if code != 0 || !strings.Contains(out, `"name": "test"`) {
		t.Fatalf("code=%d out=%q", code, out)
	}

	code, _, errOut = runApp(t, profiles, newExporterStub(), "profile", "list", "--output=xml")
	if code != 2 || !strings.Contains(errOut, "unsupported output") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}

	profiles.listFn = func(context.Context) ([]domain.Profile, error) { return nil, errors.New("disk") }
	code, _, _ = runApp(t, profiles, newExporterStub(), "profile", "list")
	if code != 1 {
		t.Fatalf("failure code = %d", code)
	}
}

func TestAppProfileDelete(t *testing.T) {
	t.Parallel()
	profiles := newUseCasesStub()
	called := ""
	profiles.deleteFn = func(_ context.Context, name string) error { called = name; return nil }
	code, out, _ := runApp(t, profiles, newExporterStub(), "profile", "delete", "--name=test")
	if code != 0 || called != "test" || !strings.Contains(out, "deleted") {
		t.Fatalf("code=%d called=%q out=%q", code, called, out)
	}

	profiles.deleteFn = func(context.Context, string) error { return domain.ErrNotFound }
	code, _, _ = runApp(t, profiles, newExporterStub(), "profile", "delete", "--name=missing")
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
}

func TestAppMetricsAndVersion(t *testing.T) {
	t.Parallel()
	exporter := newExporterStub()
	code, out, _ := runApp(t, newUseCasesStub(), exporter, "metrics")
	if code != 0 || out != "metric 1\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, out, _ = runApp(t, newUseCasesStub(), exporter, "metrics", "--format=json")
	if code != 0 || out != "{}\n" {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, _, errOut := runApp(t, newUseCasesStub(), exporter, "metrics", "--format=xml")
	if code != 2 || !strings.Contains(errOut, "unsupported metrics") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}

	exporter.prometheusFn = func(context.Context, io.Writer) error { return errors.New("boom") }
	code, _, errOut = runApp(t, newUseCasesStub(), exporter, "metrics")
	if code != 1 || !strings.Contains(errOut, "export metrics") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}

	code, out, _ = runApp(t, newUseCasesStub(), newExporterStub(), "version")
	if code != 0 || !strings.Contains(out, "cloud_cli v1") {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, out, _ = runApp(t, newUseCasesStub(), newExporterStub(), "version", "--json")
	if code != 0 || !strings.Contains(out, `"commit": "abc"`) {
		t.Fatalf("code=%d out=%q", code, out)
	}
	code, _, errOut = runApp(t, newUseCasesStub(), newExporterStub(), "version", "extra")
	if code != 2 || !strings.Contains(errOut, "unexpected arguments") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestWriteJSONFailure(t *testing.T) {
	t.Parallel()
	writer := failingWriter{}
	if err := writeJSON(writer, map[string]string{"a": "b"}); err == nil {
		t.Fatal("writeJSON() error = nil")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write failed") }

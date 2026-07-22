package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIWorkflow(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	var errOut bytes.Buffer
	app := New(t.TempDir(), &out, &errOut)

	if code := app.Run([]string{"profile", "create", "--name=test", "--user=example", "--project=new-project"}); code != 0 {
		t.Fatalf("create exit code = %d, stderr = %q", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run([]string{"profile", "get", "--name=test"}); code != 0 {
		t.Fatalf("get exit code = %d, stderr = %q", code, errOut.String())
	}
	for _, expected := range []string{"name: test", "user: example", "project: new-project"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("get output %q does not contain %q", out.String(), expected)
		}
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run([]string{"profile", "list"}); code != 0 {
		t.Fatalf("list exit code = %d, stderr = %q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "test") {
		t.Fatalf("list output = %q, want profile name", out.String())
	}

	out.Reset()
	errOut.Reset()
	if code := app.Run([]string{"profile", "delete", "--name=test"}); code != 0 {
		t.Fatalf("delete exit code = %d, stderr = %q", code, errOut.String())
	}
}

func TestHelp(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	app := New(t.TempDir(), &out, &bytes.Buffer{})
	if code := app.Run([]string{"help"}); code != 0 {
		t.Fatalf("help exit code = %d", code)
	}
	if !strings.Contains(out.String(), "profile create") {
		t.Fatalf("help output = %q", out.String())
	}
}

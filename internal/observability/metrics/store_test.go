package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shuklarituparn/cloud_cli/internal/buildinfo"
)

func TestStoreRecordAndExport(t *testing.T) {
	t.Parallel()
	store := NewStore(t.TempDir(), buildinfo.Info{Version: "1.0.0", Commit: "abc", Date: "today"})
	if err := store.Record(context.Background(), Observation{Operation: "profile.list", Duration: 25 * time.Millisecond, ProfilesTotal: new(3)}); err != nil {
		t.Fatal(err)
	}
	if err := store.Record(context.Background(), Observation{Operation: "profile.get", Duration: 5 * time.Millisecond, Err: context.Canceled}); err != nil {
		t.Fatal(err)
	}
	if err := store.Record(context.Background(), Observation{Operation: "profile.list", Duration: 50 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := store.WritePrometheus(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, expected := range []string{
		`cloud_cli_command_total{command="profile.list",status="success"} 2`,
		`cloud_cli_command_total{command="profile.get",status="failure"} 1`,
		`cloud_cli_command_duration_seconds_max{command="profile.list"} 0.05`,
		`cloud_cli_profiles_total 3`,
		`cloud_cli_build_info{version="1.0.0",commit="abc",date="today"} 1`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("metrics missing %q:\n%s", expected, text)
		}
	}

	out.Reset()
	if err := store.WriteJSON(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	var snapshot Snapshot
	if err := json.Unmarshal(out.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.ProfilesTotal != 3 || snapshot.Commands["profile.list"].SuccessCount != 2 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestStoreValidationAndCorruptState(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewStore(dir, buildinfo.Info{})
	if err := store.Record(context.Background(), Observation{}); err == nil {
		t.Fatal("empty operation accepted")
	}

	if err := os.MkdirAll(filepath.Join(dir, ".cloud_cli"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".cloud_cli", "metrics.json"), []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(context.Background()); err == nil || !strings.Contains(err.Error(), "decode metrics") {
		t.Fatalf("Snapshot() error = %v", err)
	}

	bad := Snapshot{SchemaVersion: 99, Commands: map[string]CommandStats{}}
	data, _ := json.Marshal(bad)
	if err := os.WriteFile(filepath.Join(dir, ".cloud_cli", "metrics.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Snapshot(context.Background()); err == nil || !strings.Contains(err.Error(), "unsupported metrics schema") {
		t.Fatalf("Snapshot() error = %v", err)
	}
}

func TestStoreHonorsCanceledContextWhenLocked(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := NewStore(dir, buildinfo.Info{})
	if err := os.MkdirAll(filepath.Join(dir, ".cloud_cli"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".cloud_cli", "metrics.lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := store.Snapshot(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Snapshot() error = %v", err)
	}
}

func TestQuoteLabel(t *testing.T) {
	t.Parallel()
	if got := quoteLabel("a\\b\n\"c"); got != `"a\\b\n\"c"` {
		t.Fatalf("quoteLabel() = %q", got)
	}
}

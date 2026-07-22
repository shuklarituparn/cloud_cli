package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shuklarituparn/cloud_cli/internal/buildinfo"
)

const schemaVersion = 1

type Observation struct {
	Operation     string
	Err           error
	Duration      time.Duration
	ProfilesTotal *int
}

type CommandStats struct {
	SuccessCount       uint64  `json:"success_count"`
	FailureCount       uint64  `json:"failure_count"`
	DurationSecondsSum float64 `json:"duration_seconds_sum"`
	DurationSecondsMax float64 `json:"duration_seconds_max"`
}

type Snapshot struct {
	SchemaVersion int                     `json:"schema_version"`
	UpdatedAt     time.Time               `json:"updated_at"`
	Commands      map[string]CommandStats `json:"commands"`
	ProfilesTotal int                     `json:"profiles_total"`
	Build         buildinfo.Info          `json:"build"`
}

type Store struct {
	dir       string
	statePath string
	lockPath  string
	build     buildinfo.Info
	now       func() time.Time
}

func NewStore(workingDir string, build buildinfo.Info) *Store {
	dir := filepath.Join(workingDir, ".cloud_cli")
	return &Store{
		dir:       dir,
		statePath: filepath.Join(dir, "metrics.json"),
		lockPath:  filepath.Join(dir, "metrics.lock"),
		build:     build,
		now:       time.Now,
	}
}

func (s *Store) Record(ctx context.Context, observation Observation) error {
	if observation.Operation == "" {
		return errors.New("metrics operation must not be empty")
	}
	return s.withLock(ctx, func() error {
		state, err := s.readUnlocked()
		if err != nil {
			return err
		}
		stats := state.Commands[observation.Operation]
		if observation.Err == nil {
			stats.SuccessCount++
		} else {
			stats.FailureCount++
		}
		seconds := observation.Duration.Seconds()
		stats.DurationSecondsSum += seconds
		if seconds > stats.DurationSecondsMax {
			stats.DurationSecondsMax = seconds
		}
		state.Commands[observation.Operation] = stats
		if observation.ProfilesTotal != nil {
			state.ProfilesTotal = *observation.ProfilesTotal
		}
		state.UpdatedAt = s.now().UTC()
		state.Build = s.build
		return s.writeUnlocked(state)
	})
}

func (s *Store) Snapshot(ctx context.Context) (Snapshot, error) {
	var state Snapshot
	err := s.withLock(ctx, func() error {
		var err error
		state, err = s.readUnlocked()
		return err
	})
	return state, err
}

func (s *Store) WritePrometheus(ctx context.Context, w io.Writer) error {
	state, err := s.Snapshot(ctx)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# HELP cloud_cli_command_total Total CLI command executions."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE cloud_cli_command_total counter"); err != nil {
		return err
	}
	operations := make([]string, 0, len(state.Commands))
	for operation := range state.Commands {
		operations = append(operations, operation)
	}
	sort.Strings(operations)
	for _, operation := range operations {
		stats := state.Commands[operation]
		if _, err := fmt.Fprintf(w, "cloud_cli_command_total{command=%s,status=\"success\"} %d\n", quoteLabel(operation), stats.SuccessCount); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "cloud_cli_command_total{command=%s,status=\"failure\"} %d\n", quoteLabel(operation), stats.FailureCount); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "# HELP cloud_cli_command_duration_seconds_sum Total command execution time."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE cloud_cli_command_duration_seconds_sum counter"); err != nil {
		return err
	}
	for _, operation := range operations {
		stats := state.Commands[operation]
		if _, err := fmt.Fprintf(w, "cloud_cli_command_duration_seconds_sum{command=%s} %s\n", quoteLabel(operation), strconv.FormatFloat(stats.DurationSecondsSum, 'g', -1, 64)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "# HELP cloud_cli_command_duration_seconds_max Maximum observed command execution time."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE cloud_cli_command_duration_seconds_max gauge"); err != nil {
		return err
	}
	for _, operation := range operations {
		stats := state.Commands[operation]
		if _, err := fmt.Fprintf(w, "cloud_cli_command_duration_seconds_max{command=%s} %s\n", quoteLabel(operation), strconv.FormatFloat(stats.DurationSecondsMax, 'g', -1, 64)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "# HELP cloud_cli_profiles_total Number of profiles observed by the latest list operation."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE cloud_cli_profiles_total gauge"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "cloud_cli_profiles_total %d\n", state.ProfilesTotal); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# HELP cloud_cli_build_info Build metadata."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE cloud_cli_build_info gauge"); err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "cloud_cli_build_info{version=%s,commit=%s,date=%s} 1\n", quoteLabel(state.Build.Version), quoteLabel(state.Build.Commit), quoteLabel(state.Build.Date))
	return err
}

func (s *Store) WriteJSON(ctx context.Context, w io.Writer) error {
	state, err := s.Snapshot(ctx)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(state)
}

func (s *Store) readUnlocked() (Snapshot, error) {
	state := Snapshot{SchemaVersion: schemaVersion, Commands: map[string]CommandStats{}, Build: s.build}
	data, err := os.ReadFile(s.statePath)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("read metrics state: %w", err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return Snapshot{}, fmt.Errorf("decode metrics state: %w", err)
	}
	if state.Commands == nil {
		state.Commands = map[string]CommandStats{}
	}
	if state.SchemaVersion != schemaVersion {
		return Snapshot{}, fmt.Errorf("unsupported metrics schema version %d", state.SchemaVersion)
	}
	state.Build = s.build
	return state, nil
}

func (s *Store) writeUnlocked(state Snapshot) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("create metrics directory: %w", err)
	}
	temp, err := os.CreateTemp(s.dir, "metrics-*.tmp")
	if err != nil {
		return fmt.Errorf("create metrics temp file: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		_ = temp.Close()
		return fmt.Errorf("encode metrics state: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync metrics state: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close metrics state: %w", err)
	}
	if err := os.Rename(tempName, s.statePath); err != nil {
		return fmt.Errorf("replace metrics state: %w", err)
	}
	return nil
}

func (s *Store) withLock(ctx context.Context, fn func() error) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("create metrics directory: %w", err)
	}
	deadline := s.now().Add(2 * time.Second)
	for {
		file, err := os.OpenFile(s.lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = file.Close()
			defer os.Remove(s.lockPath)
			return fn()
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("acquire metrics lock: %w", err)
		}
		if info, statErr := os.Stat(s.lockPath); statErr == nil && s.now().Sub(info.ModTime()) > 10*time.Second {
			_ = os.Remove(s.lockPath)
			continue
		}
		if s.now().After(deadline) {
			return errors.New("timed out acquiring metrics lock")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func quoteLabel(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return "\"" + value + "\""
}

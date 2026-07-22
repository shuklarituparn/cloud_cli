package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	application "github.com/shuklarituparn/cloud_cli/internal/application/profile"
	"github.com/shuklarituparn/cloud_cli/internal/buildinfo"
	"github.com/shuklarituparn/cloud_cli/internal/infrastructure/filestore"
	"github.com/shuklarituparn/cloud_cli/internal/interfaces/cli"
	"github.com/shuklarituparn/cloud_cli/internal/observability/intrumentation"
	"github.com/shuklarituparn/cloud_cli/internal/observability/metrics"
)

func main() {
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot determine current directory: %v\n", err)
		os.Exit(1)
	}

	logger := newLogger(os.Getenv("CLOUD_CLI_LOG_FORMAT"))
	build := buildinfo.Current()
	repository := filestore.NewRepository(workingDir, filestore.YAMLCodec{})
	service := application.NewService(repository)
	metricsStore := metrics.NewStore(workingDir, build)
	instrumentedService := intrumentation.NewService(service, metricsStore, logger)
	app := cli.New(instrumentedService, metricsStore, build, os.Stdout, os.Stderr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(app.Run(ctx, os.Args[1:]))
}

func newLogger(format string) *slog.Logger {
	options := &slog.HandlerOptions{Level: slog.LevelWarn}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(os.Stderr, options))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, options))
}

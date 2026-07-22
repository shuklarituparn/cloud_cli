package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	application "github.com/shuklarituparn/cloud_cli/internal/application/profile"
	"github.com/shuklarituparn/cloud_cli/internal/buildinfo"
	domain "github.com/shuklarituparn/cloud_cli/internal/domain/profile"
)

const usage = `cloud_cli - cloud profile management CLI

Usage:
  cloud_cli profile create --name=<name> --user=<user> --project=<project>
  cloud_cli profile get --name=<name> [--output=yaml|json]
  cloud_cli profile list [--output=table|json]
  cloud_cli profile delete --name=<name>
  cloud_cli metrics [--format=prometheus|json]
  cloud_cli version
  cloud_cli help

Commands:
  profile create  Create <name>.yaml in the current directory
  profile get     Read one profile
  profile list    List profiles in the current directory
  profile delete  Delete one profile
  metrics         Export persistent low-cardinality CLI metrics
  version         Print build metadata
  help            Show this help
`

type MetricsExporter interface {
	WritePrometheus(context.Context, io.Writer) error
	WriteJSON(context.Context, io.Writer) error
}

type App struct {
	profiles application.UseCases
	metrics  MetricsExporter
	build    buildinfo.Info
	out      io.Writer
	errOut   io.Writer
}

func New(profiles application.UseCases, metrics MetricsExporter, build buildinfo.Info, out, errOut io.Writer) *App {
	return &App{profiles: profiles, metrics: metrics, build: build, out: out, errOut: errOut}
}

func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.printUsage(a.errOut)
		return 2
	}
	switch args[0] {
	case "help", "-h", "--help":
		a.printUsage(a.out)
		return 0
	case "profile":
		return a.runProfile(ctx, args[1:])
	case "metrics":
		return a.runMetrics(ctx, args[1:])
	case "version":
		return a.runVersion(args[1:])
	default:
		fmt.Fprintf(a.errOut, "unknown command %q\n\n", args[0])
		a.printUsage(a.errOut)
		return 2
	}
}

func (a *App) runProfile(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.errOut, "profile subcommand is required")
		return 2
	}
	switch args[0] {
	case "create":
		return a.create(ctx, args[1:])
	case "get":
		return a.get(ctx, args[1:])
	case "list":
		return a.list(ctx, args[1:])
	case "delete":
		return a.delete(ctx, args[1:])
	default:
		fmt.Fprintf(a.errOut, "unknown profile subcommand %q\n", args[0])
		return 2
	}
}

func (a *App) create(ctx context.Context, args []string) int {
	flags := a.newFlagSet("profile create")
	name := flags.String("name", "", "profile name")
	user := flags.String("user", "", "cloud user")
	project := flags.String("project", "", "cloud project")
	if err := parseFlags(flags, args, a.errOut); err != nil {
		return 2
	}
	p, err := a.profiles.Create(ctx, application.CreateCommand{Name: *name, User: *user, Project: *project})
	if err != nil {
		return a.writeOperationError("create profile", err)
	}
	fmt.Fprintf(a.out, "profile %q created\n", p.Name)
	return 0
}

func (a *App) get(ctx context.Context, args []string) int {
	flags := a.newFlagSet("profile get")
	name := flags.String("name", "", "profile name")
	output := flags.String("output", "yaml", "output format: yaml or json")
	if err := parseFlags(flags, args, a.errOut); err != nil {
		return 2
	}
	p, err := a.profiles.Get(ctx, *name)
	if err != nil {
		return a.writeOperationError("get profile", err)
	}
	switch *output {
	case "yaml":
		fmt.Fprintf(a.out, "name: %s\nuser: %s\nproject: %s\n", p.Name, p.User, p.Project)
	case "json":
		if err := writeJSON(a.out, profileDTO(p)); err != nil {
			fmt.Fprintf(a.errOut, "write output: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(a.errOut, "unsupported output format %q\n", *output)
		return 2
	}
	return 0
}

func (a *App) list(ctx context.Context, args []string) int {
	flags := a.newFlagSet("profile list")
	output := flags.String("output", "table", "output format: table or json")
	if err := parseFlags(flags, args, a.errOut); err != nil {
		return 2
	}
	profiles, err := a.profiles.List(ctx)
	if err != nil {
		return a.writeOperationError("list profiles", err)
	}
	switch *output {
	case "table":
		writer := tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "NAME\tUSER\tPROJECT")
		for _, p := range profiles {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", p.Name, p.User, p.Project)
		}
		if err := writer.Flush(); err != nil {
			fmt.Fprintf(a.errOut, "write output: %v\n", err)
			return 1
		}
	case "json":
		items := make([]map[string]string, 0, len(profiles))
		for _, p := range profiles {
			items = append(items, profileDTO(p))
		}
		if err := writeJSON(a.out, items); err != nil {
			fmt.Fprintf(a.errOut, "write output: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(a.errOut, "unsupported output format %q\n", *output)
		return 2
	}
	return 0
}

func (a *App) delete(ctx context.Context, args []string) int {
	flags := a.newFlagSet("profile delete")
	name := flags.String("name", "", "profile name")
	if err := parseFlags(flags, args, a.errOut); err != nil {
		return 2
	}
	if err := a.profiles.Delete(ctx, *name); err != nil {
		return a.writeOperationError("delete profile", err)
	}
	fmt.Fprintf(a.out, "profile %q deleted\n", *name)
	return 0
}

func (a *App) runMetrics(ctx context.Context, args []string) int {
	flags := a.newFlagSet("metrics")
	format := flags.String("format", "prometheus", "output format: prometheus or json")
	if err := parseFlags(flags, args, a.errOut); err != nil {
		return 2
	}
	var err error
	switch *format {
	case "prometheus":
		err = a.metrics.WritePrometheus(ctx, a.out)
	case "json":
		err = a.metrics.WriteJSON(ctx, a.out)
	default:
		fmt.Fprintf(a.errOut, "unsupported metrics format %q\n", *format)
		return 2
	}
	if err != nil {
		fmt.Fprintf(a.errOut, "export metrics: %v\n", err)
		return 1
	}
	return 0
}

func (a *App) runVersion(args []string) int {
	flags := a.newFlagSet("version")
	jsonOutput := flags.Bool("json", false, "print build metadata as JSON")
	if err := parseFlags(flags, args, a.errOut); err != nil {
		return 2
	}
	if *jsonOutput {
		if err := writeJSON(a.out, a.build); err != nil {
			fmt.Fprintf(a.errOut, "write output: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(a.out, "cloud_cli %s (commit=%s, built=%s)\n", a.build.Version, a.build.Commit, a.build.Date)
	return 0
}

func (a *App) writeOperationError(operation string, err error) int {
	fmt.Fprintf(a.errOut, "%s: %v\n", operation, err)
	if errors.Is(err, domain.ErrAlreadyExists) || errors.Is(err, domain.ErrNotFound) {
		return 1
	}
	if errors.Is(err, domain.ErrInvalidName) || errors.Is(err, domain.ErrInvalidUser) || errors.Is(err, domain.ErrInvalidProject) {
		return 2
	}
	return 1
}

func (a *App) newFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(a.errOut)
	flags.Usage = func() { fmt.Fprintf(a.errOut, "usage: cloud_cli %s [flags]\n", name); flags.PrintDefaults() }
	return flags
}

func (a *App) printUsage(w io.Writer) { fmt.Fprint(w, usage) }

func parseFlags(flags *flag.FlagSet, args []string, errOut io.Writer) error {
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		err := fmt.Errorf("unexpected arguments: %v", flags.Args())
		fmt.Fprintln(errOut, err)
		return err
	}
	return nil
}

func profileDTO(p domain.Profile) map[string]string {
	return map[string]string{"name": p.Name.String(), "user": p.User, "project": p.Project}
}
func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

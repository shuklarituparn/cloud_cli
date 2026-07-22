package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/shuklarituparn/cloud_cli/internal/observability/profile"
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
`

type App struct {
	store *profile.Store
	out   io.Writer
	err   io.Writer
}

func New(dir string, out, errOut io.Writer) *App {
	return &App{
		store: profile.NewStore(dir),
		out:   out,
		err:   errOut,
	}
}

func (a *App) Run(args []string) int {
	if len(args) == 0 {
		a.printUsage(a.err)
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		a.printUsage(a.out)
		return 0
	case "profile":
		return a.runProfile(args[1:])
	default:
		fmt.Fprintf(a.err, "unknown command %q\n\n", args[0])
		a.printUsage(a.err)
		return 2
	}
}

func (a *App) runProfile(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(a.err, "profile subcommand is required")
		return 2
	}

	switch args[0] {
	case "create":
		return a.create(args[1:])
	case "get":
		return a.get(args[1:])
	case "list":
		return a.list(args[1:])
	case "delete":
		return a.delete(args[1:])
	default:
		fmt.Fprintf(a.err, "unknown profile subcommand %q\n", args[0])
		return 2
	}
}

func (a *App) create(args []string) int {
	flags := a.newFlagSet("profile create")
	name := flags.String("name", "", "profile name")
	user := flags.String("user", "", "cloud user")
	project := flags.String("project", "", "cloud project")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(a.err, "unexpected arguments: %v\n", flags.Args())
		return 2
	}

	err := a.store.Create(profile.Profile{Name: *name, User: *user, Project: *project})
	if err != nil {
		fmt.Fprintf(a.err, "create profile: %v\n", err)
		if errors.Is(err, profile.ErrAlreadyExists) {
			return 1
		}
		return 2
	}

	fmt.Fprintf(a.out, "profile %q created\n", *name)
	return 0
}

func (a *App) get(args []string) int {
	flags := a.newFlagSet("profile get")
	name := flags.String("name", "", "profile name")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(a.err, "unexpected arguments: %v\n", flags.Args())
		return 2
	}

	p, err := a.store.Get(*name)
	if err != nil {
		fmt.Fprintf(a.err, "get profile: %v\n", err)
		if errors.Is(err, profile.ErrNotFound) {
			return 1
		}
		return 2
	}

	fmt.Fprintf(a.out, "name: %s\nuser: %s\nproject: %s\n", p.Name, p.User, p.Project)
	return 0
}

func (a *App) list(args []string) int {
	flags := a.newFlagSet("profile list")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(a.err, "unexpected arguments: %v\n", flags.Args())
		return 2
	}

	profiles, err := a.store.List()
	if err != nil {
		fmt.Fprintf(a.err, "list profiles: %v\n", err)
		return 1
	}

	writer := tabwriter.NewWriter(a.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tUSER\tPROJECT")
	for _, p := range profiles {
		fmt.Fprintf(writer, "%s\t%s\t%s\n", p.Name, p.User, p.Project)
	}
	if err := writer.Flush(); err != nil {
		fmt.Fprintf(a.err, "write output: %v\n", err)
		return 1
	}
	return 0
}

func (a *App) delete(args []string) int {
	flags := a.newFlagSet("profile delete")
	name := flags.String("name", "", "profile name")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(a.err, "unexpected arguments: %v\n", flags.Args())
		return 2
	}

	if err := a.store.Delete(*name); err != nil {
		fmt.Fprintf(a.err, "delete profile: %v\n", err)
		if errors.Is(err, profile.ErrNotFound) {
			return 1
		}
		return 2
	}

	fmt.Fprintf(a.out, "profile %q deleted\n", *name)
	return 0
}

func (a *App) newFlagSet(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(a.err)
	flags.Usage = func() {
		fmt.Fprintf(a.err, "usage: cloud_cli %s [flags]\n", name)
		flags.PrintDefaults()
	}
	return flags
}

func (a *App) printUsage(w io.Writer) {
	fmt.Fprint(w, usage)
}

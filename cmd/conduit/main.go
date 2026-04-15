package main

import (
	"fmt"
	"os"

	"github.com/Crowley723/conduit/internal/commands"
	"github.com/Crowley723/conduit/internal/version"
	"github.com/alecthomas/kong"
)

var CLI struct {
	commands.Globals

	Version bool `short:"v" help:"Print version information"`

	Serve   commands.ServeCommand   `cmd:"" default:"1" help:"Run conduit server"`
	Migrate commands.MigrateCommand `cmd:"" help:"Database migration commands"`
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("conduit"),
		kong.Description("Conduit application"),
		kong.UsageOnError(),
	)

	if CLI.Version {
		fmt.Printf("Version: %s\n", version.GetVersion())
		fmt.Printf("Git Commit: %s\n", version.GetGitCommit())
		fmt.Printf("Build Time: %s\n", version.GetBuildTime())
		os.Exit(0)
	}

	if err := ctx.Run(&CLI.Globals); err != nil {
		os.Exit(1)
	}
}

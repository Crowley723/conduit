package main

import (
	"fmt"
	"os"

	"github.com/Crowley723/conduit/internal/devtools"
	"github.com/alecthomas/kong"
)

var CLI struct {
	Up     devtools.UpCommand     `cmd:"" help:"Start development infrastructure"`
	Down   devtools.DownCommand   `cmd:"" help:"Stop development infrastructure"`
	Status devtools.StatusCommand `cmd:"" help:"Show development infrastructure status"`
	Logs   devtools.LogsCommand   `cmd:"" help:"Tail service logs"`
	Serve  devtools.ServeCommand  `cmd:"" help:"Start backend and frontend with hot-reload"`
}

func main() {
	ctx := kong.Parse(&CLI,
		kong.Name("conduit-dev"),
		kong.Description("Conduit development environment tool"),
		kong.UsageOnError(),
	)

	if err := ctx.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// takeout-audit exposes the audit workflow without upload/delete/admin commands.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/dhcgn/immich-admin-cli/internal/commands"
	"github.com/urfave/cli/v3"
)

var version = "dev"
var commit = "none"
var date = "unknown"

func command() *cli.Command {
	root := commands.TakeoutAudit()
	root.Name = "takeout-audit"
	root.Version = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
	root.Flags = append(root.Flags, &cli.StringFlag{Name: "config", Value: "config.audit.yaml", Usage: "private YAML config; not needed for --inventory-only"})
	return root
}

func main() {
	if err := command().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if code, ok := errors.AsType[cli.ExitCoder](err); ok {
			os.Exit(code.ExitCode())
		}
		os.Exit(2)
	}
}

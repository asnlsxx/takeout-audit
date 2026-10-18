package commands

import (
	"context"
	"fmt"
	"github.com/dhcgn/immich-admin-cli/internal/client"
	"github.com/urfave/cli/v3"
)

// Server exposes the version operation used by the audit workflow natively.
func Server() *cli.Command {
	return &cli.Command{Name: "server", Usage: "Server information", Commands: []*cli.Command{{
		Name: "version", Usage: "Show server version (GET /server/version)",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			c, err := newClient(ctx, cmd)
			if err != nil {
				return err
			}
			v, err := c.ServerVersion(ctx)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.Writer, client.FormatServerVersion(v))
			return nil
		},
	}}}
}

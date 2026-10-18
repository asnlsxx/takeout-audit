package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/dhcgn/immich-admin-cli/internal/client"
	"github.com/dhcgn/immich-admin-cli/internal/takeoutaudit"
	"github.com/dhcgn/immich-admin-cli/internal/workflows"
	"github.com/urfave/cli/v3"
)

func auditTakeoutCommand() *cli.Command {
	limits := takeoutaudit.DefaultOptions()
	return &cli.Command{
		Name:      "audit-takeout",
		Usage:     "Audit every Takeout ZIP/directory entry against Immich catalog checksums; never upload or delete",
		ArgsUsage: "ZIP|DIR [ZIP|DIR ...]",
		Description: "Experimental, contract-tested for stable Immich 3.2.x; live-server validation is pending. " +
			"Reads all supplied inputs without extraction. Writes ledger.jsonl, media.csv and summary.json in a NEW directory. " +
			"Does not verify Google export completeness, metadata, external libraries or current server file integrity. " +
			"API key needs user.read and asset.upload. No asset upload is performed. Exit 0: scoped audit complete; 1: differences/attention; 2: incomplete (including inventory-only).",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "out", Required: true, Usage: "NEW report directory, outside inputs; parent must exist"},
			&cli.BoolFlag{Name: "inventory-only", Usage: "offline scan with no config/API; media stays unknown and exit status is 2"},
			&cli.IntFlag{Name: "max-entries", Value: limits.MaxEntries, Usage: "maximum file entries across all inputs"},
			&cli.Int64Flag{Name: "max-file-bytes", Value: limits.MaxFileBytes, Usage: "maximum uncompressed bytes per file"},
			&cli.Int64Flag{Name: "max-total-bytes", Value: limits.MaxTotalBytes, Usage: "maximum total uncompressed bytes read"},
		},
		Action: clientWorkflowAuditTakeout,
	}
}

// TakeoutAudit exposes only the read-only audit action for the dedicated binary.
func TakeoutAudit() *cli.Command { return auditTakeoutCommand() }

func clientWorkflowAuditTakeout(ctx context.Context, cmd *cli.Command) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	paths := cmd.Args().Slice()
	options := takeoutaudit.Options{MaxEntries: cmd.Int("max-entries"), MaxFileBytes: cmd.Int64("max-file-bytes"), MaxTotalBytes: cmd.Int64("max-total-bytes")}
	lastProgress := time.Now()
	options.Progress = func(entries int, bytesRead int64) {
		if time.Since(lastProgress) >= 2*time.Second {
			fmt.Fprintf(os.Stderr, "Scanned %d entries; read %.1f GiB...\n", entries, float64(bytesRead)/(1<<30))
			lastProgress = time.Now()
		}
	}
	if len(paths) == 0 {
		return cli.Exit("pass one or more ZIP files or directories", 2)
	}
	if err := options.Validate(); err != nil {
		return cli.Exit(err, 2)
	}
	paths, err := takeoutaudit.ValidateInputs(paths)
	if err != nil {
		return cli.Exit(err, 2)
	}
	out, err := takeoutaudit.CreateOutput(cmd.String("out"), paths)
	if err != nil {
		return cli.Exit(err, 2)
	}
	fmt.Fprintln(os.Stderr, "Scanning supplied inputs; report:", out)
	for _, path := range paths {
		fmt.Fprintf(os.Stderr, "  input %q\n", path)
	}
	report, err := takeoutaudit.Scan(ctx, paths, options)
	if err != nil {
		return cli.Exit(err, 2)
	}
	report.Summary.InventoryOnly = cmd.Bool("inventory-only")
	if !report.Summary.InventoryOnly && ctx.Err() == nil {
		connectCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		c, connectErr := newClient(connectCtx, cmd, client.WithAuditOnly())
		cancel()
		if connectErr != nil {
			report.Summary.Errors = append(report.Summary.Errors, "client_setup_failed; check config and server origin")
		} else {
			fmt.Fprintln(os.Stderr, "Checking unique media content against account catalog...")
			workflows.AuditTakeoutRemote(ctx, c, report)
		}
	}
	if ctx.Err() != nil {
		report.Summary.Errors = append(report.Summary.Errors, "cancelled")
	}
	if err := report.Write(out); err != nil {
		return cli.Exit(fmt.Errorf("report write failed: %w", err), 2)
	}
	s := report.Summary
	fmt.Fprintf(os.Stdout, "%d source entries; %d media occurrences; %d distinct content groups; %d extra source copies\nContent groups: matched=%d; not matched=%d; trashed=%d; unknown=%d\nSource read errors=%d\nReport: %s\n",
		s.Entries, s.MediaOccurrences, s.ContentGroups, s.DuplicateMediaCopies, s.GroupStatuses[takeoutaudit.Match], s.GroupStatuses[takeoutaudit.NotMatched], s.GroupStatuses[takeoutaudit.Trashed], s.GroupStatuses[takeoutaudit.Unknown], s.Statuses[takeoutaudit.ReadError], out)
	fmt.Fprintln(os.Stdout, "Catalog comparison only. Never use this report alone to delete originals.")
	if code := report.ExitCode(); code != 0 {
		return cli.Exit("inspect summary.json: attention required or audit incomplete", code)
	}
	return nil
}

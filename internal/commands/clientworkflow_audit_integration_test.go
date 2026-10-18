//go:build integration

package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dhcgn/immich-admin-cli/internal/client"
	"github.com/dhcgn/immich-admin-cli/internal/config"
	"github.com/dhcgn/immich-admin-cli/internal/takeoutaudit"
	"github.com/dhcgn/immich-admin-cli/internal/workflows"
)

// This smoke test only queries staging. The separate release gate in the guide
// also requires seeded matched/missing/trashed assets in a disposable instance.
func TestAuditTakeoutReadOnlyIntegration(t *testing.T) {
	if _, err := os.Stat(stagingConfigPath); err != nil {
		t.Skip("no isolated staging configuration")
	}
	cfg, err := config.Load(stagingConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	c, err := client.New(cfg, client.WithAuditOnly())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	path := filepath.Join(t.TempDir(), "takeout.zip")
	auditCommandFixture(t, path)
	r, err := takeoutaudit.Scan(ctx, []string{path}, takeoutaudit.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	workflows.AuditTakeoutRemote(ctx, c, r)
	if !r.Summary.RunComplete {
		t.Fatalf("real server audit incomplete: %+v", r.Summary)
	}
	if r.Summary.AccountID == "" {
		t.Fatal("account not recorded")
	}
}

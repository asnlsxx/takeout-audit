//go:build integration

package commands

import (
	"archive/zip"
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dhcgn/immich-admin-cli/internal/client"
	"github.com/dhcgn/immich-admin-cli/internal/config"
	"github.com/dhcgn/immich-admin-cli/internal/immichapi"
	"github.com/dhcgn/immich-admin-cli/internal/takeoutaudit"
	"github.com/dhcgn/immich-admin-cli/internal/workflows"
	"github.com/google/uuid"
)

// Only the opt-in disposable runner may seed/trash fixtures. The audit itself
// uses a separate key with user.read + asset.upload and the transport allowlist.
func TestAuditTakeoutLifecycleIntegration(t *testing.T) {
	if os.Getenv("TAKEOUT_AUDIT_DISPOSABLE") != "1" {
		t.Skip("requires explicit disposable-instance opt-in")
	}
	cfg, err := config.Load(stagingConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(cfg.Server)
	if err != nil || u.Hostname() != "127.0.0.1" {
		t.Fatal("lifecycle test only accepts loopback disposable instances")
	}
	queryKey := os.Getenv("TAKEOUT_AUDIT_QUERY_KEY")
	if queryKey == "" {
		t.Fatal("missing restricted query key")
	}
	admin, err := client.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	query, err := client.New(&config.Config{Server: cfg.Server, APIKey: queryKey}, client.WithAuditOnly())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dir := t.TempDir()
	var uploaded []uuid.UUID
	t.Cleanup(func() {
		if len(uploaded) == 0 {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		force := true
		resp, err := admin.API.DeleteAssetsWithResponse(cleanupCtx, immichapi.AssetBulkDeleteDto{Ids: uploaded, Force: &force})
		if err == nil {
			err = client.Check(resp, http.StatusNoContent)
		}
		if err != nil {
			t.Errorf("fixture cleanup failed: %v", err)
		}
	})
	names := []string{"present.png", "trashed.png", "absent.png"}
	var bodies [][]byte
	for i, name := range names {
		path := filepath.Join(dir, name)
		bodies = append(bodies, writeRandomPNG(t, path))
		if i < 2 {
			id, err := workflows.UploadAssetFile(ctx, admin, path, workflows.UploadOptions{})
			if err != nil {
				t.Fatal(err)
			}
			uploaded = append(uploaded, id)
		}
	}
	force := false
	resp, err := admin.API.DeleteAssetsWithResponse(ctx, immichapi.AssetBulkDeleteDto{Ids: []uuid.UUID{uploaded[1]}, Force: &force})
	if err == nil {
		err = client.Check(resp, http.StatusNoContent)
	}
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		info, err := admin.API.GetAssetInfoWithResponse(ctx, uploaded[1], nil)
		if err == nil && info.JSON200 != nil && info.JSON200.IsTrashed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fixture did not enter trash")
		}
		time.Sleep(200 * time.Millisecond)
	}
	statistics := func() immichapi.AssetStatsResponseDto {
		s, err := admin.API.GetAssetStatisticsWithResponse(ctx, nil)
		if err == nil {
			err = client.Check(s, http.StatusOK)
		}
		if err != nil || s.JSON200 == nil {
			t.Fatalf("statistics unavailable: %v", err)
		}
		return *s.JSON200
	}
	before := statistics()
	archivePath := filepath.Join(t.TempDir(), "takeout.zip")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for i, name := range names {
		entry, err := w.Create("Photos/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(bodies[i]); err != nil {
			t.Fatal(err)
		}
	}
	entry, err := w.Create("Album/present.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(bodies[0]); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := takeoutaudit.Scan(ctx, []string{archivePath}, takeoutaudit.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	workflows.AuditTakeoutRemote(ctx, query, r)
	if !r.Summary.RunComplete || r.Summary.GroupStatuses[takeoutaudit.Match] != 1 || r.Summary.GroupStatuses[takeoutaudit.Trashed] != 1 || r.Summary.GroupStatuses[takeoutaudit.NotMatched] != 1 || r.Summary.DuplicateMediaCopies != 1 {
		t.Fatalf("wrong real lifecycle result: %+v", r.Summary)
	}
	if after := statistics(); after != before {
		t.Fatalf("audit changed statistics: before=%+v after=%+v", before, after)
	}
	for i, id := range uploaded {
		info, err := admin.API.GetAssetInfoWithResponse(ctx, id, nil)
		if err != nil || info.JSON200 == nil || info.JSON200.IsTrashed != (i == 1) {
			t.Fatal("audit changed asset state")
		}
	}
	t.Logf("real Immich %s: matched=1, trashed=1, absent=1, extra source copies=1; restricted query key; no audit mutations", r.Summary.ServerVersion)
}

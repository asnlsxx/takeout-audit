package workflows

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dhcgn/immich-admin-cli/internal/client"
	"github.com/dhcgn/immich-admin-cli/internal/config"
	"github.com/dhcgn/immich-admin-cli/internal/immichapi"
	"github.com/dhcgn/immich-admin-cli/internal/takeoutaudit"
	"github.com/google/uuid"
)

func testAuditReport() *takeoutaudit.Report {
	return &takeoutaudit.Report{Entries: []takeoutaudit.Entry{
		{ID: "s1:0", Path: "private-name.jpg", Kind: "media", SHA1: "a", SHA256: "aa", GroupID: "g1", Status: takeoutaudit.Unknown},
		{ID: "s1:1", Kind: "media", SHA1: "a", SHA256: "aa", GroupID: "g1", Status: takeoutaudit.Unknown},
		{ID: "s1:2", Kind: "media", SHA1: "b", SHA256: "bb", GroupID: "g2", Status: takeoutaudit.Unknown},
		{ID: "s1:3", Kind: "media", SHA1: "c", SHA256: "cc", GroupID: "g3", Status: takeoutaudit.Unknown},
	}}
}

func TestAuditRemoteContract(t *testing.T) {
	for _, mode := range []string{"ok", "missing", "extra", "duplicate", "trash_missing", "bad_action", "auth", "old_version", "prerelease", "no_identity", "null_version", "null_identity", "incomplete_version", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			checks := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Header.Get("x-api-key") != "test-secret" {
					t.Error("auth header missing")
				}
				switch r.Method + " " + r.URL.Path {
				case "GET /api/users/me":
					if mode == "no_identity" {
						w.Write([]byte(`{}`))
						return
					}
					if mode == "null_identity" {
						w.Write([]byte(`null`))
						return
					}
					w.Write([]byte(`{"id":"11111111-1111-4111-8111-111111111111"}`))
				case "GET /api/server/version":
					if mode == "incomplete_version" {
						w.Write([]byte(`{"major":3,"minor":2}`))
						return
					}
					if mode == "old_version" {
						w.Write([]byte(`{"major":2,"minor":4,"patch":0}`))
						return
					}
					if mode == "prerelease" {
						w.Write([]byte(`{"major":3,"minor":2,"patch":0,"prerelease":1}`))
						return
					}
					if mode == "null_version" {
						w.Write([]byte(`null`))
						return
					}
					w.Write([]byte(`{"major":3,"minor":2,"patch":0,"prerelease":null}`))
				case "POST /api/assets/bulk-upload-check":
					checks++
					var request immichapi.AssetBulkUploadCheckDto
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
					}
					if len(request.Assets) != 3 {
						t.Errorf("expected 3 content groups, got %d", len(request.Assets))
					}
					for _, v := range request.Assets {
						if strings.Contains(v.Id, "private") {
							t.Error("filename leaked")
						}
					}
					if mode == "auth" {
						w.WriteHeader(403)
						w.Write([]byte(`{"error":"test-secret"}`))
						return
					}
					results := []map[string]any{
						{"id": "g3", "action": "reject", "reason": "duplicate", "assetId": "33333333-3333-4333-8333-333333333333", "isTrashed": true},
						{"id": "g2", "action": "accept"},
						{"id": "g1", "action": "reject", "reason": "duplicate", "assetId": "22222222-2222-4222-8222-222222222222", "isTrashed": false},
					}
					switch mode {
					case "missing":
						results = results[:2]
					case "extra":
						results = append(results, map[string]any{"id": "unexpected", "action": "accept"})
					case "duplicate":
						results[2] = results[0]
					case "trash_missing":
						delete(results[2], "isTrashed")
					case "bad_action":
						results[2]["action"] = "unexpected"
					}
					json.NewEncoder(w).Encode(map[string]any{"results": results})
				default:
					t.Errorf("unexpected or mutating request %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			defer server.Close()
			c, err := client.New(&config.Config{Server: server.URL, APIKey: "test-secret"}, client.WithAuditOnly())
			if err != nil {
				t.Fatal(err)
			}
			report := testAuditReport()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			AuditTakeoutRemote(ctx, c, report)
			if mode == "ok" {
				if report.Summary.Statuses[takeoutaudit.Match] != 2 || report.Summary.Statuses[takeoutaudit.Trashed] != 1 || report.Summary.Statuses[takeoutaudit.NotMatched] != 1 || !report.Summary.RunComplete {
					t.Fatalf("bad results %+v", report.Summary)
				}
				if checks != 1 {
					t.Fatal("duplicate query")
				}
			} else if report.Summary.RunComplete || report.Summary.Statuses[takeoutaudit.Match] != 0 {
				t.Fatalf("failure became success %+v", report.Summary)
			}
			blob, _ := json.Marshal(report)
			if strings.Contains(string(blob), "test-secret") {
				t.Fatal("response body leaked secret")
			}
		})
	}
}

func TestAuditRejectsIncompleteDuplicate(t *testing.T) {
	id := uuid.New()
	trash := false
	reason := immichapi.AssetRejectReasonDuplicate
	base := immichapi.AssetBulkUploadCheckResult{Action: immichapi.Reject, Id: "one", AssetId: &id, Reason: &reason, IsTrashed: &trash}
	status, _, _ := classifyAuditResult(base)
	if status != takeoutaudit.Match {
		t.Fatal("valid duplicate rejected")
	}
	base.IsTrashed = nil
	status, _, _ = classifyAuditResult(base)
	if status != takeoutaudit.Unknown {
		t.Fatal("missing trash accepted")
	}
}

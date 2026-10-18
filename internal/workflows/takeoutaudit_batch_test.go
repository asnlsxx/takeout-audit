package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dhcgn/immich-admin-cli/internal/client"
	"github.com/dhcgn/immich-admin-cli/internal/config"
	"github.com/dhcgn/immich-admin-cli/internal/immichapi"
	"github.com/dhcgn/immich-admin-cli/internal/takeoutaudit"
)

func TestAuditBatchesAndRetries(t *testing.T) {
	requests, checked := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/users/me":
			w.Write([]byte(`{"id":"11111111-1111-4111-8111-111111111111"}`))
		case "/api/server/version":
			w.Write([]byte(`{"major":3,"minor":2,"patch":0}`))
		case "/api/assets/bulk-upload-check":
			requests++
			if requests == 1 {
				w.WriteHeader(429)
				return
			}
			var body immichapi.AssetBulkUploadCheckDto
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body.Assets) > 200 || len(body.Assets) == 0 {
				t.Errorf("bad batch size %d", len(body.Assets))
			}
			checked += len(body.Assets)
			results := make([]immichapi.AssetBulkUploadCheckResult, len(body.Assets))
			for i, item := range body.Assets {
				results[i] = immichapi.AssetBulkUploadCheckResult{Id: item.Id, Action: immichapi.Accept}
			}
			json.NewEncoder(w).Encode(immichapi.AssetBulkUploadCheckResponseDto{Results: results})
		default:
			t.Error("unexpected endpoint")
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	c, err := client.New(&config.Config{Server: server.URL, APIKey: "test"}, client.WithAuditOnly())
	if err != nil {
		t.Fatal(err)
	}
	r := &takeoutaudit.Report{}
	for i := range 401 {
		r.Entries = append(r.Entries, takeoutaudit.Entry{ID: fmt.Sprint(i), GroupID: fmt.Sprint(i), Kind: "media", SHA1: fmt.Sprint(i), Status: takeoutaudit.Unknown})
	}
	AuditTakeoutRemote(context.Background(), c, r)
	if requests != 4 || checked != 401 || r.Summary.Statuses[takeoutaudit.NotMatched] != 401 || !r.Summary.RunComplete {
		t.Fatalf("bad batches/retry: requests=%d checked=%d summary=%+v", requests, checked, r.Summary)
	}
}

package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dhcgn/immich-admin-cli/internal/config"
	"github.com/dhcgn/immich-admin-cli/internal/immichapi"
)

func TestAuditTransportBlocksWritesAndRedirects(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Location", "/api/assets")
		w.WriteHeader(302)
	}))
	defer server.Close()
	c, err := New(&config.Config{Server: server.URL, APIKey: "fixture"}, WithAuditOnly())
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.API.DeleteAssetsWithResponse(context.Background(), immichapi.AssetBulkDeleteDto{})
	if err == nil || hits != 0 {
		t.Fatal("destructive request escaped guard")
	}
	resp, err := c.API.GetMyUserWithResponse(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hits != 1 || resp.StatusCode() != 302 {
		t.Fatal("redirect followed")
	}
}

func TestAuditOriginValidation(t *testing.T) {
	for _, s := range []string{"https://user:pass@example.org", "https://example.org/api", "https://example.org?secret=x", "https://example.org#fragment"} {
		if _, err := New(&config.Config{Server: s, APIKey: "fixture"}, WithAuditOnly()); err == nil {
			t.Errorf("accepted %q", s)
		}
	}
}

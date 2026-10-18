package commands

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dhcgn/immich-admin-cli/internal/immichapi"
	"github.com/dhcgn/immich-admin-cli/internal/takeoutaudit"
	"github.com/urfave/cli/v3"
)

func auditCommandFixture(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 42, A: 255})
	var pngData bytes.Buffer
	if err := png.Encode(&pngData, img); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for _, name := range []string{"Takeout/Google Photos/2025/photo.png", "Takeout/Google Photos/Album/photo.png"} {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(pngData.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	entry, err := w.Create("Takeout/Google Photos/2025/photo.png.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, `{"title":"photo.png"}`); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAuditTakeoutCommandEndToEnd(t *testing.T) {
	for _, offline := range []bool{false, true} {
		t.Run(map[bool]string{false: "mock_server", true: "offline"}[offline], func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "takeout.zip")
			auditCommandFixture(t, input)
			before, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			hits := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				w.Header().Set("Content-Type", "application/json")
				switch r.Method + " " + r.URL.Path {
				case "GET /api/users/me":
					io.WriteString(w, `{"id":"11111111-1111-4111-8111-111111111111","email":"fixture@example.invalid"}`)
				case "GET /api/server/version":
					io.WriteString(w, `{"major":3,"minor":2,"patch":0,"prerelease":null}`)
				case "POST /api/assets/bulk-upload-check":
					var req immichapi.AssetBulkUploadCheckDto
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
					}
					if len(req.Assets) != 1 {
						t.Errorf("wanted one content query: %d", len(req.Assets))
						w.WriteHeader(400)
						return
					}
					json.NewEncoder(w).Encode(map[string]any{"results": []map[string]any{{"id": req.Assets[0].Id, "action": "reject", "reason": "duplicate", "assetId": "22222222-2222-4222-8222-222222222222", "isTrashed": false}}})
				default:
					t.Errorf("unexpected write/request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			defer server.Close()
			cfg := filepath.Join(dir, "config.yaml")
			if !offline {
				if err := os.WriteFile(cfg, []byte("server: "+server.URL+"\napi_key: synthetic-test-key\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("IMMICH_SERVER", server.URL)
			t.Setenv("IMMICH_API_KEY", "synthetic-test-key")
			out := filepath.Join(dir, "report")
			root := &cli.Command{Name: "test", Writer: io.Discard, ErrWriter: io.Discard, Flags: []cli.Flag{&cli.StringFlag{Name: "config", Value: cfg}}, Commands: []*cli.Command{ClientWorkflow()}, ExitErrHandler: func(context.Context, *cli.Command, error) {}}
			args := []string{"test", "cw", "audit-takeout", "--out", out}
			if offline {
				args = append(args, "--inventory-only")
			}
			args = append(args, input)
			err = root.Run(context.Background(), args)
			if offline {
				var exit cli.ExitCoder
				if !errors.As(err, &exit) || exit.ExitCode() != 2 {
					t.Fatalf("offline exit: %v", err)
				}
				if hits != 0 {
					t.Fatal("offline made network calls")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(out, "summary.json"))
			if err != nil {
				t.Fatal(err)
			}
			var summary takeoutaudit.Summary
			if err := json.Unmarshal(data, &summary); err != nil {
				t.Fatal(err)
			}
			if summary.Entries != 3 || summary.ContentGroups != 1 || summary.RunComplete == offline {
				t.Fatalf("bad summary: %+v", summary)
			}
			if strings.Contains(string(data), "synthetic-test-key") {
				t.Fatal("secret in report")
			}
			after, err := os.ReadFile(input)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("input modified")
			}
		})
	}
}

func TestAuditSkillContainsUsage(t *testing.T) {
	text := buildSkillText([]*cli.Command{ClientWorkflow(), Server()})
	for _, required := range []string{"audit-takeout", "--inventory-only", "--max-file-bytes", "--out", "server version"} {
		if !strings.Contains(text, required) {
			t.Errorf("skill missing %s", required)
		}
	}
}

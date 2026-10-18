package takeoutaudit

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDuplicateInputsFailBeforeReading(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.zip")
	writeZip(t, path, fixture{"photo.png", pngBytes(t, 9), 0})
	called := false
	options := DefaultOptions()
	options.Progress = func(int, int64) { called = true }
	if _, err := Scan(context.Background(), []string{path, path}, options); err == nil || called {
		t.Fatal("duplicate inputs must fail before media I/O")
	}
	if _, err := ValidateInputs([]string{dir, path}); err == nil {
		t.Fatal("overlapping directory and ZIP accepted")
	}
	hardlink := filepath.Join(dir, "alias.zip")
	if err := os.Link(path, hardlink); err == nil {
		if _, err := ValidateInputs([]string{path, hardlink}); err == nil {
			t.Fatal("duplicate physical source accepted")
		}
	}
}

func TestNonUTF8NameHasLosslessRepresentation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.zip")
	name := string([]byte{0xff, 0xfe}) + ".png"
	writeZip(t, path, fixture{name, pngBytes(t, 11), 0})
	r, err := Scan(context.Background(), []string{path}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(r.Entries[0])
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip Entry
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(roundtrip.PathRawBase64)
	if err != nil || string(raw) != name {
		t.Fatal("legacy name bytes lost")
	}
}

func TestDistinctGroupCountsAndSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.zip")
	body := pngBytes(t, 12)
	writeZip(t, path, fixture{"a.png", body, 0}, fixture{"album/a.png", body, 0})
	r, err := Scan(context.Background(), []string{path}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for i := range r.Entries {
		r.Entries[i].Status = Match
	}
	r.Finalize()
	if r.Summary.GroupStatuses[Match] != 1 || r.Summary.Statuses[Match] != 2 || r.Summary.DuplicateMediaCopies != 1 || r.Summary.Sources[0].Entries != 2 || r.Summary.BytesRead != int64(len(body)*2) {
		t.Fatalf("incorrect accounting: %+v", r.Summary)
	}
}

func BenchmarkScanTakeout5000Entries(b *testing.B) {
	path := filepath.Join(b.TempDir(), "many.zip")
	f, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	w := zip.NewWriter(f)
	for i := range 5000 {
		entry, err := w.Create(fmt.Sprintf("photos/%05d.jpg", i))
		if err != nil {
			b.Fatal(err)
		}
		if _, err := entry.Write(bytes.Repeat([]byte{byte(i)}, 512)); err != nil {
			b.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		b.Fatal(err)
	}
	if err := f.Close(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		r, err := Scan(context.Background(), []string{path}, DefaultOptions())
		if err != nil || len(r.Entries) != 5000 {
			b.Fatalf("scan: %v", err)
		}
	}
}

func FuzzMalformedZIPNeverClaimsComplete(f *testing.F) {
	f.Add([]byte("not a zip"))
	f.Add([]byte("PK\x03\x04"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 128*1024 {
			t.Skip()
		}
		path := filepath.Join(t.TempDir(), "input.zip")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		options := DefaultOptions()
		options.MaxEntries = 100
		options.MaxFileBytes = 1 << 20
		options.MaxTotalBytes = 2 << 20
		r, err := Scan(context.Background(), []string{path}, options)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.Summary.Sources) != 1 {
			t.Fatal("input disappeared")
		}
		if r.Summary.Sources[0].Error != "" && r.Summary.RunComplete {
			t.Fatal("failed source was marked complete")
		}
		for _, entry := range r.Entries {
			if entry.Status == Match {
				t.Fatal("local-only scan invented a remote match")
			}
		}
	})
}

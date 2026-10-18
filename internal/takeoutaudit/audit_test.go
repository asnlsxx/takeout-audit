package takeoutaudit

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixture struct {
	name string
	body []byte
	mode os.FileMode
}

func pngBytes(t *testing.T, red uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: red, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
func writeZip(t *testing.T, path string, entries ...fixture) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: zip.Store}
		if e.mode != 0 {
			h.SetMode(e.mode)
		}
		writer, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(e.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestScanAccountsForEveryOccurrence(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.zip"), filepath.Join(dir, "b.zip")
	one, two := pngBytes(t, 10), pngBytes(t, 20)
	writeZip(t, a, fixture{"Photos/image.png", one, 0}, fixture{"Album/image.png", one, 0}, fixture{"Photos/image.png", two, 0}, fixture{".hidden.png", one, 0}, fixture{"image.png.json", []byte(`{"title":"image.png"}`), 0}, fixture{"other.txt", []byte("notes"), 0}, fixture{"link.png", []byte("target"), os.ModeSymlink | 0o777})
	writeZip(t, b, fixture{"Photos/image.png", one, 0}, fixture{"clip.MOV", []byte("synthetic movie bytes; not a decoder test"), 0})
	r, err := Scan(context.Background(), []string{a, b}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Entries) != 9 || r.Summary.ContentGroups != 3 || r.Summary.HashedMedia != 6 {
		t.Fatalf("unexpected ledger: %+v", r.Summary)
	}
	if r.Summary.MediaOccurrences != r.Summary.HashedMedia+r.Summary.MediaReadErrors+r.Summary.ExcludedMedia {
		t.Fatal("media accounting failed")
	}
	if r.Entries[0].Copies != 4 || r.Entries[2].GroupID == r.Entries[0].GroupID || r.Entries[0].ID == r.Entries[2].ID {
		t.Fatal("duplicate names/content conflated")
	}
	if r.Entries[6].Status != Excluded || r.Entries[4].Status != Metadata || r.Entries[5].Status != Unclassified {
		t.Fatal("missing non-media accounting")
	}
	if r.ExitCode() != 2 {
		t.Fatal("unqueried inventory must not pass")
	}
	var total int
	for _, count := range r.Summary.Statuses {
		total += count
	}
	if total != len(r.Entries) {
		t.Fatal("status conservation failed")
	}
}

func TestCorruptMemberCannotMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.zip")
	payload := pngBytes(t, 40)
	writeZip(t, path, fixture{"image.png", payload, 0})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	index := bytes.Index(data, payload)
	if index < 0 {
		t.Fatal("fixture payload not found")
	}
	data[index+5] ^= 1
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Scan(context.Background(), []string{path}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if r.Entries[0].Status != ReadError || r.Entries[0].SHA1 != "" || r.Summary.RunComplete {
		t.Fatalf("corruption accepted: %+v", r)
	}
}

func TestArchiveFailureCancellationAndLimits(t *testing.T) {
	dir := t.TempDir()
	bad, good := filepath.Join(dir, "bad.zip"), filepath.Join(dir, "good.zip")
	if err := os.WriteFile(bad, []byte("not zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeZip(t, good, fixture{"a.png", pngBytes(t, 1), 0}, fixture{"b.png", pngBytes(t, 2), 0})
	r, _ := Scan(context.Background(), []string{bad, good}, DefaultOptions())
	if r.Summary.Sources[0].EnumerationComplete || len(r.Entries) != 2 {
		t.Fatal("bad source hid next source or was counted complete")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, _ = Scan(ctx, []string{good}, DefaultOptions())
	if r.Summary.RunComplete || r.Summary.Sources[0].EnumerationComplete {
		t.Fatal("cancel passed")
	}
	o := DefaultOptions()
	o.MaxEntries = 1
	r, _ = Scan(context.Background(), []string{good}, o)
	if len(r.Entries) != 1 || r.Summary.Sources[0].EnumerationComplete {
		t.Fatal("entry limit ignored")
	}
	o = DefaultOptions()
	o.MaxFileBytes = 1
	r, _ = Scan(context.Background(), []string{good}, o)
	if r.Summary.Statuses[ReadError] != 2 || r.Summary.HashedMedia != 0 {
		t.Fatal("size limit ignored")
	}
	o = DefaultOptions()
	o.MaxTotalBytes = 1
	r, _ = Scan(context.Background(), []string{good}, o)
	if r.Summary.Statuses[ReadError] != 2 {
		t.Fatal("total limit ignored")
	}
}

func TestExtractedDirectoryAndReport(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input")
	if err := os.Mkdir(input, 0o700); err != nil {
		t.Fatal(err)
	}
	name := "=公式,照片.png"
	if err := os.WriteFile(filepath.Join(input, name), pngBytes(t, 3), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateOutput(filepath.Join(input, "report"), []string{input}); err == nil {
		t.Fatal("nested report accepted")
	}
	r, err := Scan(context.Background(), []string{input}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	r.Entries[0].Status = Match
	r.Entries[0].Reason = "catalog only"
	out, err := CreateOutput(filepath.Join(dir, "report"), []string{input})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Write(out); err != nil {
		t.Fatal(err)
	}
	if r.ExitCode() != 0 {
		t.Fatal("successful scoped audit should pass")
	}
	if _, err := CreateOutput(out, []string{input}); err == nil {
		t.Fatal("overwrote directory")
	}
	data, err := os.ReadFile(filepath.Join(out, "ledger.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var e Entry
	if err := json.Unmarshal(bytes.TrimSpace(data), &e); err != nil {
		t.Fatal(err)
	}
	if e.Path != name {
		t.Fatal("JSONL path lost")
	}
	data, err = os.ReadFile(filepath.Join(out, "media.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if rows[1][2] != "'"+name {
		t.Fatalf("CSV formula not escaped: %q", rows[1][2])
	}
	r.Entries[0].Status = Trashed
	r.Finalize()
	if r.ExitCode() != 1 {
		t.Fatal("trash should need attention")
	}
	r.Summary.InventoryOnly = true
	r.Finalize()
	if r.ExitCode() != 2 {
		t.Fatal("offline should be incomplete")
	}
}

func TestLocalHashCollision(t *testing.T) {
	r := Report{Entries: []Entry{{Kind: "media", Size: 10, SHA1: "same", SHA256: "a", Status: Unknown}, {Kind: "media", Size: 10, SHA1: "same", SHA256: "b", Status: Unknown}}}
	r.Group()
	for _, e := range r.Entries {
		if e.Reason != "local_sha1_collision" || e.Copies != 1 {
			t.Fatal("collision not isolated")
		}
	}
}

func TestCSVHostileCells(t *testing.T) {
	for _, v := range []string{"=SUM(1)", "  +cmd", "-1", "@formula", "\tfoo", "\nfoo"} {
		if !strings.HasPrefix(safeCell(v), "'") {
			t.Errorf("unsafe %q", v)
		}
	}
	if safeCell("中文.png") != "中文.png" {
		t.Fatal("unnecessary CSV transformation")
	}
}

func TestHiddenAndDuplicateInputs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".photo.png"), pngBytes(t, 1), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Scan(context.Background(), []string{dir}, DefaultOptions())
	if err != nil || len(r.Entries) != 1 {
		t.Fatal("hidden input skipped")
	}
	if _, err := Scan(context.Background(), []string{dir, dir}, DefaultOptions()); err == nil {
		t.Fatal("duplicate input accepted")
	}
}

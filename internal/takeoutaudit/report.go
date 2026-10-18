package takeoutaudit

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// CreateOutput reserves a new report directory before doing any expensive work.
// It rejects output nested inside an input tree, including symlink aliases.
func CreateOutput(path string, inputs []string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", fmt.Errorf("output parent must already exist: %w", err)
	}
	abs = filepath.Join(parent, filepath.Base(abs))
	for _, input := range inputs {
		info, err := os.Stat(input)
		if err != nil || !info.IsDir() {
			continue
		}
		root, err := filepath.EvalSymlinks(input)
		if err != nil {
			return "", err
		}
		root, err = filepath.Abs(root)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(root, abs)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return "", fmt.Errorf("output must be outside input directory %q", input)
		}
	}
	if err := os.Mkdir(abs, 0o700); err != nil {
		return "", fmt.Errorf("create new output directory: %w", err)
	}
	return abs, nil
}

// Write writes summary last: its presence means the two ledgers were closed
// successfully. The caller must pass a freshly reserved directory.
func (r *Report) Write(dir string) error {
	r.Finalize()
	f, err := os.OpenFile(filepath.Join(dir, "ledger.jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, e := range r.Entries {
		if err := enc.Encode(e); err != nil {
			f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	f, err = os.OpenFile(filepath.Join(dir, "media.csv"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	if err := w.Write([]string{"id", "source_id", "path", "bytes", "group_id", "source_copies", "status", "reason", "asset_id", "sha1_base64", "sha256"}); err != nil {
		f.Close()
		return err
	}
	for _, e := range r.Entries {
		if e.Kind != "media" {
			continue
		}
		row := []string{e.ID, e.SourceID, e.Path, strconv.FormatInt(e.Size, 10), e.GroupID, strconv.Itoa(e.Copies), string(e.Status), e.Reason, e.AssetID, e.SHA1, e.SHA256}
		for i := range row {
			row[i] = safeCell(row[i])
		}
		if err := w.Write(row); err != nil {
			f.Close()
			return err
		}
	}
	w.Flush()
	err = w.Error()
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	f, err = os.OpenFile(filepath.Join(dir, "summary.json"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	enc = json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(r.Summary)
	closeErr = f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func safeCell(s string) string {
	trimmed := strings.TrimLeftFunc(s, unicode.IsSpace)
	if strings.ContainsAny(s, "\t\r\n") || (trimmed != "" && strings.ContainsAny(trimmed[:1], "=+-@")) {
		return "'" + s
	}
	return s
}

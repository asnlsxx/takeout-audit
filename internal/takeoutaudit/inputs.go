package takeoutaudit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateInputs rejects duplicates and overlapping roots before reading media.
// Unreadable/missing inputs are left for Scan to account for in its source ledger.
func ValidateInputs(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("pass one or more ZIP files or extracted directories")
	}
	type input struct {
		absolute, canonical string
		info                os.FileInfo
	}
	var prior []input
	var output []string
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		canonical, err := filepath.EvalSymlinks(abs)
		if err != nil {
			canonical = abs
		}
		info, _ := os.Stat(abs)
		for _, previous := range prior {
			samePath := canonical == previous.canonical
			if filepath.Separator == '\\' {
				samePath = strings.EqualFold(canonical, previous.canonical)
			}
			if samePath || (info != nil && previous.info != nil && os.SameFile(info, previous.info)) {
				return nil, fmt.Errorf("duplicate input %q and %q", previous.absolute, abs)
			}
			if (info != nil && info.IsDir() && inside(canonical, previous.canonical)) || (previous.info != nil && previous.info.IsDir() && inside(previous.canonical, canonical)) {
				return nil, fmt.Errorf("overlapping inputs %q and %q; supply each source only once", previous.absolute, abs)
			}
		}
		prior = append(prior, input{abs, canonical, info})
		output = append(output, abs)
	}
	return output, nil
}

func inside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

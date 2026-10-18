package takeoutaudit

import (
	"archive/zip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

type scanner struct {
	ctx     context.Context
	options Options
	report  *Report
	bytes   int64
	buffer  []byte
}

// Scan always returns the observed ledger, even on cancellation or a corrupt
// archive. Enumeration errors live on their source and prevent a complete run.
func Scan(ctx context.Context, paths []string, options Options) (*Report, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("pass one or more ZIP files or extracted directories")
	}
	if err := options.Validate(); err != nil {
		return nil, err
	}
	paths, err := ValidateInputs(paths)
	if err != nil {
		return nil, err
	}
	r := &Report{Summary: Summary{SchemaVersion: 1, Started: time.Now().UTC(), Errors: []string{},
		ContractValidation: "mock_contract_tests_only; real_server_validation_pending",
		Limitations:        []string{"Only supplied inputs are covered; Google export completeness is unknown.", "Matches describe account catalog checksums, not current server disk integrity.", "Metadata, albums and external libraries are not verified.", "Not matched does not mean lost: editing, re-encoding or another account can change results."}}}
	s := scanner{ctx: ctx, options: options, report: r, buffer: make([]byte, 128*1024)}
	for _, path := range paths {
		abs := path
		source := Source{ID: fmt.Sprintf("s%04d", len(r.Summary.Sources)+1), Path: abs}
		start := len(r.Entries)
		info, err := os.Lstat(abs)
		switch {
		case err != nil:
			source.Error = err.Error()
		case info.Mode()&os.ModeSymlink != 0:
			source.Error = "input symlinks are not followed"
		case info.IsDir():
			err = s.directory(abs, &source)
		case !info.Mode().IsRegular():
			source.Error = "input is not a regular file or directory"
		case !strings.EqualFold(filepath.Ext(abs), ".zip"):
			source.Error = "file input must be an ordinary ZIP; use a directory for extracted files"
		default:
			err = s.archive(abs, &source)
			after, statErr := os.Stat(abs)
			if statErr != nil || !os.SameFile(info, after) || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
				err = fmt.Errorf("source archive changed during scan")
				for i := start; i < len(r.Entries); i++ {
					r.Entries[i].Status = ReadError
					r.Entries[i].Reason = err.Error()
					r.Entries[i].SHA1 = ""
					r.Entries[i].SHA256 = ""
				}
			}
		}
		if err != nil {
			source.Error = err.Error()
		}
		if source.Error == "" {
			source.EnumerationComplete = true
		}
		source.Entries = len(r.Entries) - start
		r.Summary.Sources = append(r.Summary.Sources, source)
	}
	r.Summary.BytesRead = s.bytes
	if ctx.Err() != nil {
		r.Summary.Errors = append(r.Summary.Errors, "cancelled")
	}
	r.Group()
	r.Finalize()
	return r, nil
}

func classify(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		return "metadata"
	}
	const media = ".jpg .jpeg .jpe .png .gif .webp .heic .heif .avif .tif .tiff .bmp .dng .raw .cr2 .cr3 .nef .nrw .arw .raf .rw2 .orf .pef .srw .x3f .3fr .fff .iiq .rwl .mp4 .mov .m4v .avi .3gp .3g2 .mpg .mpeg .mkv .webm .mts .m2ts .wmv .vob .flv .insv .360 .jxl .psd"
	if ext != "" && strings.Contains(" "+media+" ", " "+ext+" ") {
		return "media"
	}
	return "other"
}

func (s *scanner) newEntry(source *Source, name string, member int, size int64) (*Entry, error) {
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	if len(s.report.Entries) >= s.options.MaxEntries {
		return nil, fmt.Errorf("max-entries limit reached; remaining entries not enumerated")
	}
	s.report.Entries = append(s.report.Entries, Entry{ID: fmt.Sprintf("%s:%d", source.ID, member), SourceID: source.ID, Path: name, Member: member, Kind: classify(name), Size: size, Status: Unknown, Reason: "not_queried"})
	e := &s.report.Entries[len(s.report.Entries)-1]
	if !utf8.ValidString(name) {
		e.PathRawBase64 = base64.StdEncoding.EncodeToString([]byte(name))
	}
	if s.options.Progress != nil {
		s.options.Progress(len(s.report.Entries), s.bytes)
	}
	return e, nil
}

func (s *scanner) read(e *Entry, open func() (io.ReadCloser, error)) {
	if e.Size < 0 || e.Size > s.options.MaxFileBytes || e.Size > s.options.MaxTotalBytes-s.bytes {
		e.Status, e.Reason = ReadError, "uncompressed_byte_limit"
		return
	}
	f, err := open()
	if err != nil {
		e.Status, e.Reason = ReadError, err.Error()
		return
	}
	limit := min(s.options.MaxFileBytes, s.options.MaxTotalBytes-s.bytes)
	n, a, b, err := hashStream(s.ctx, f, limit, s.buffer)
	closeErr := f.Close()
	s.bytes += n
	if err == nil {
		err = closeErr
	}
	if err == nil && n != e.Size {
		err = fmt.Errorf("size changed: expected %d, read %d", e.Size, n)
	}
	if err != nil {
		e.Status, e.Reason = ReadError, err.Error()
		return
	}
	switch e.Kind {
	case "media":
		e.SHA1, e.SHA256 = a, b
	case "metadata":
		e.Status, e.Reason = Metadata, "sidecar_semantics_not_checked"
	default:
		e.Status, e.Reason = Unclassified, "extension_not_in_media_allowlist"
	}
}

func (s *scanner) archive(path string, source *Source) error {
	f, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("opening ZIP: %w", err)
	}
	defer f.Close()
	for i, member := range f.File {
		if member.FileInfo().IsDir() {
			continue
		}
		size := int64(member.UncompressedSize64)
		e, err := s.newEntry(source, member.Name, i, size)
		if err != nil {
			return err
		}
		if len(member.Name) > 32768 {
			e.Status, e.Reason = Excluded, "path_too_long"
			continue
		}
		if !member.Mode().IsRegular() {
			e.Status, e.Reason = Excluded, "non_regular_member"
			continue
		}
		if member.Flags&1 != 0 {
			e.Status, e.Reason = ReadError, "encrypted_zip_not_supported"
			continue
		}
		s.read(e, member.Open)
	}
	return nil
}

func (s *scanner) directory(path string, source *Source) error {
	member := 0
	return filepath.WalkDir(path, func(name string, d fs.DirEntry, walkErr error) error {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(path, name)
		if err != nil {
			return err
		}
		if walkErr != nil {
			e, entryErr := s.newEntry(source, filepath.ToSlash(rel), member, 0)
			if entryErr != nil {
				return entryErr
			}
			member++
			e.Status, e.Reason = ReadError, walkErr.Error()
			// A directory failure may hide arbitrarily many descendants.
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		e, err := s.newEntry(source, filepath.ToSlash(rel), member, info.Size())
		if err != nil {
			return err
		}
		member++
		if !info.Mode().IsRegular() {
			e.Status, e.Reason = Excluded, "non_regular_file"
			return nil
		}
		s.read(e, func() (io.ReadCloser, error) {
			// OpenRoot prevents symlink races from escaping this input tree.
			root, err := os.OpenRoot(path)
			if err != nil {
				return nil, err
			}
			defer root.Close()
			return root.Open(rel)
		})
		after, err := os.Lstat(name)
		if err != nil || !os.SameFile(info, after) || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
			e.Status, e.Reason, e.SHA1, e.SHA256 = ReadError, "source_changed_during_read", "", ""
		}
		return nil
	})
}

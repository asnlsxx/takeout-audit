// Package takeoutaudit inventories local Takeout files without extracting or
// changing them. Remote queries live in workflows; this package has no network.
package takeoutaudit

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

type Status string

const (
	Match        Status = "catalog_match"
	Trashed      Status = "trashed"
	NotMatched   Status = "not_matched"
	Unknown      Status = "unknown"
	ReadError    Status = "read_error"
	Metadata     Status = "metadata_not_checked"
	Unclassified Status = "unclassified"
	Excluded     Status = "excluded"
)

type Entry struct {
	ID            string `json:"id"`
	SourceID      string `json:"source_id"`
	Path          string `json:"path"`
	PathRawBase64 string `json:"path_raw_base64,omitempty"`
	Member        int    `json:"member_index"`
	Kind          string `json:"kind"`
	Size          int64  `json:"size"`
	SHA1          string `json:"sha1_base64,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
	GroupID       string `json:"group_id,omitempty"`
	Copies        int    `json:"source_copies,omitempty"`
	Status        Status `json:"status"`
	Reason        string `json:"reason,omitempty"`
	AssetID       string `json:"asset_id,omitempty"`
}

type Source struct {
	ID                  string `json:"id"`
	Path                string `json:"path"`
	EnumerationComplete bool   `json:"enumeration_complete"`
	Error               string `json:"error,omitempty"`
	Entries             int    `json:"entries"`
}

type Summary struct {
	SchemaVersion        int            `json:"schema_version"`
	Started              time.Time      `json:"started"`
	Finished             time.Time      `json:"finished"`
	Sources              []Source       `json:"sources"`
	Entries              int            `json:"entries"`
	MediaOccurrences     int            `json:"media_occurrences"`
	HashedMedia          int            `json:"hashed_media"`
	MediaReadErrors      int            `json:"media_read_errors"`
	ExcludedMedia        int            `json:"excluded_media"`
	ContentGroups        int            `json:"content_groups"`
	DuplicateMediaCopies int            `json:"duplicate_media_copies"`
	BytesRead            int64          `json:"bytes_read"`
	GroupStatuses        map[Status]int `json:"group_statuses"`
	Statuses             map[Status]int `json:"statuses"`
	RunComplete          bool           `json:"run_complete"`
	NeedsAttention       bool           `json:"needs_attention"`
	InventoryOnly        bool           `json:"inventory_only"`
	AccountID            string         `json:"account_id,omitempty"`
	ServerVersion        string         `json:"server_version,omitempty"`
	Server               string         `json:"server,omitempty"`
	ContractValidation   string         `json:"contract_validation"`
	Errors               []string       `json:"errors"`
	Limitations          []string       `json:"limitations"`
}

type Report struct {
	Entries []Entry
	Summary Summary
}

type Options struct {
	MaxEntries    int
	MaxFileBytes  int64
	MaxTotalBytes int64
	Progress      func(entries int, bytesRead int64)
}

func DefaultOptions() Options {
	return Options{MaxEntries: 500000, MaxFileBytes: 64 << 30, MaxTotalBytes: 2 << 40}
}

func (o Options) Validate() error {
	if o.MaxEntries <= 0 || o.MaxFileBytes <= 0 || o.MaxTotalBytes <= 0 || o.MaxFileBytes >= 1<<63-1 {
		return fmt.Errorf("scan limits must be positive; max-file-bytes must be below MaxInt64")
	}
	return nil
}

// Finalize can be repeated after remote results have been applied.
func (r *Report) Finalize() {
	s := &r.Summary
	s.Entries, s.MediaOccurrences, s.HashedMedia, s.ContentGroups = len(r.Entries), 0, 0, 0
	s.MediaReadErrors, s.ExcludedMedia = 0, 0
	s.Statuses = make(map[Status]int)
	s.GroupStatuses = make(map[Status]int)
	s.RunComplete = !s.InventoryOnly && len(s.Errors) == 0
	s.NeedsAttention = false
	groups := make(map[string]Status)
	for _, source := range s.Sources {
		if !source.EnumerationComplete || source.Error != "" {
			s.RunComplete = false
		}
	}
	for _, e := range r.Entries {
		s.Statuses[e.Status]++
		if e.Kind == "media" {
			s.MediaOccurrences++
			if e.Status == ReadError {
				s.MediaReadErrors++
			}
			if e.Status == Excluded {
				s.ExcludedMedia++
			}
			if e.SHA1 != "" {
				s.HashedMedia++
			}
		}
		if e.GroupID != "" {
			if prior, exists := groups[e.GroupID]; exists && prior != e.Status {
				groups[e.GroupID] = Unknown
				s.RunComplete = false
			} else {
				groups[e.GroupID] = e.Status
			}
		}
		if e.Status == Unknown || e.Status == ReadError {
			s.RunComplete = false
		}
		if e.Status != Match && e.Status != Metadata {
			s.NeedsAttention = true
		}
	}
	s.ContentGroups = len(groups)
	s.DuplicateMediaCopies = max(0, s.HashedMedia-s.ContentGroups)
	for _, status := range groups {
		s.GroupStatuses[status]++
	}
	if s.MediaOccurrences == 0 || !s.RunComplete {
		s.NeedsAttention = true
	}
	s.Finished = time.Now().UTC()
}

func (r *Report) ExitCode() int {
	if !r.Summary.RunComplete {
		return 2
	}
	if r.Summary.NeedsAttention {
		return 1
	}
	return 0
}

// Group uses SHA256 in addition to SHA1 and size: SHA1 is only the server
// protocol key. A local SHA1 collision must never become a catalog match.
func (r *Report) Group() {
	counts := make(map[string]int)
	sha1s := make(map[string]string)
	collisions := make(map[string]bool)
	for i := range r.Entries {
		e := &r.Entries[i]
		if e.Kind != "media" || e.SHA1 == "" {
			continue
		}
		key := fmt.Sprintf("%s:%d", e.SHA256, e.Size)
		counts[key]++
		if prev, ok := sha1s[e.SHA1]; ok && prev != key {
			collisions[e.SHA1] = true
		}
		sha1s[e.SHA1] = key
	}
	// Walk source order instead of map order for deterministic group IDs.
	groupIDs := make(map[string]string)
	for i := range r.Entries {
		e := &r.Entries[i]
		if e.Kind != "media" || e.SHA1 == "" {
			continue
		}
		key := fmt.Sprintf("%s:%d", e.SHA256, e.Size)
		id, ok := groupIDs[key]
		if !ok {
			id = fmt.Sprintf("g%08d", len(groupIDs)+1)
			groupIDs[key] = id
		}
		e.GroupID, e.Copies = id, counts[key]
		if collisions[e.SHA1] {
			e.Status, e.Reason = Unknown, "local_sha1_collision"
		}
	}
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func hashStream(ctx context.Context, reader io.Reader, limit int64, buffer []byte) (int64, string, string, error) {
	a, b := sha1.New(), sha256.New()
	n, err := io.CopyBuffer(io.MultiWriter(a, b), io.LimitReader(contextReader{ctx, reader}, limit+1), buffer)
	if err != nil {
		return n, "", "", err
	}
	if n > limit {
		return n, "", "", fmt.Errorf("uncompressed byte limit exceeded")
	}
	return n, base64.StdEncoding.EncodeToString(a.Sum(nil)), hex.EncodeToString(b.Sum(nil)), nil
}

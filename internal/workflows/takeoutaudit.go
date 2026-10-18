package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dhcgn/immich-admin-cli/internal/client"
	"github.com/dhcgn/immich-admin-cli/internal/immichapi"
	"github.com/dhcgn/immich-admin-cli/internal/takeoutaudit"
	"github.com/google/uuid"
)

// AuditTakeoutRemote only fills successful media hashes. No filenames are sent.
// Errors deliberately omit server response bodies, which can echo credentials.
func AuditTakeoutRemote(ctx context.Context, c *client.Client, report *takeoutaudit.Report) {
	defer report.Finalize()
	report.Summary.Server = c.Server
	fail := func(reason string) {
		report.Summary.Errors = append(report.Summary.Errors, reason)
		for i := range report.Entries {
			e := &report.Entries[i]
			if e.Status == takeoutaudit.Unknown && e.Reason != "local_sha1_collision" {
				e.Reason = reason
			}
		}
	}
	// All remote operations have a bounded context, unlike large downloads.
	checkCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	me, err := c.API.GetMyUserWithResponse(checkCtx)
	cancel()
	if err != nil {
		fail("account_query_failed")
		return
	}
	if err := client.Check(me, http.StatusOK); err != nil || me.JSON200 == nil || me.JSON200.Id == uuid.Nil {
		fail("account_unverified")
		return
	}
	report.Summary.AccountID = me.JSON200.Id.String()
	checkCtx, cancel = context.WithTimeout(ctx, 45*time.Second)
	version, err := c.API.GetServerVersionWithResponse(checkCtx)
	cancel()
	if err != nil {
		fail("version_query_failed")
		return
	}
	if err := client.Check(version, http.StatusOK); err != nil || version.JSON200 == nil {
		fail("version_unverified")
		return
	}
	// Required numeric fields must actually be present; generated non-pointer
	// ints alone cannot distinguish an omitted patch from patch zero.
	var fields map[string]json.RawMessage
	if json.Unmarshal(version.Body, &fields) != nil {
		fail("version_unverified")
		return
	}
	for _, key := range []string{"major", "minor", "patch"} {
		if raw, ok := fields[key]; !ok || string(raw) == "null" {
			fail("version_incomplete")
			return
		}
	}
	v := *version.JSON200
	report.Summary.ServerVersion = client.FormatServerVersion(v)
	// The local contract suite covers 3.2.x only. This is not a claim of live
	// validation; the report explicitly records that release gate as pending.
	if v.Major != 3 || v.Minor != 2 || v.Patch < 0 || v.Prerelease != nil {
		fail("unsupported_version_requires_stable_3.2.x")
		return
	}
	groups := make(map[string][]int)
	var items []immichapi.AssetBulkUploadCheckItem
	for i, e := range report.Entries {
		if e.Kind != "media" || e.SHA1 == "" || e.Status != takeoutaudit.Unknown || e.Reason == "local_sha1_collision" {
			continue
		}
		if _, ok := groups[e.GroupID]; !ok {
			items = append(items, immichapi.AssetBulkUploadCheckItem{Id: e.GroupID, Checksum: e.SHA1})
		}
		groups[e.GroupID] = append(groups[e.GroupID], i)
	}
	for start := 0; start < len(items); start += 200 {
		batch := items[start:min(start+200, len(items))]
		results, reason := auditBatch(ctx, c, batch)
		if reason == "" {
			reason = validateAuditResults(batch, results)
		}
		if reason != "" {
			report.Summary.Errors = append(report.Summary.Errors, fmt.Sprintf("batch_%d:%s", start/200+1, reason))
			for _, item := range batch {
				for _, index := range groups[item.Id] {
					report.Entries[index].Reason = reason
				}
			}
			continue
		}
		for _, result := range results {
			status, why, assetID := classifyAuditResult(result)
			for _, index := range groups[result.Id] {
				e := &report.Entries[index]
				e.Status, e.Reason, e.AssetID = status, why, assetID
			}
		}
	}
}

func auditBatch(ctx context.Context, c *client.Client, items []immichapi.AssetBulkUploadCheckItem) ([]immichapi.AssetBulkUploadCheckResult, string) {
	for attempt := range 3 {
		requestCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		resp, err := c.API.CheckBulkUploadWithResponse(requestCtx, immichapi.AssetBulkUploadCheckDto{Assets: items})
		cancel()
		if ctx.Err() != nil {
			return nil, "cancelled"
		}
		if err == nil && resp != nil && resp.StatusCode() == http.StatusOK {
			if err := client.Check(resp, http.StatusOK); err != nil || resp.JSON200 == nil {
				return nil, "missing_json_response"
			}
			return resp.JSON200.Results, ""
		}
		reason, retry := "bulk_check_transport_error", true
		if err == nil && resp != nil {
			reason = fmt.Sprintf("bulk_check_http_%d", resp.StatusCode())
			retry = resp.StatusCode() == 429 || resp.StatusCode() >= 500
		}
		if !retry || attempt == 2 {
			return nil, reason
		}
		timer := time.NewTimer(time.Duration(attempt+1) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, "cancelled"
		case <-timer.C:
		}
	}
	return nil, "bulk_check_failed"
}

func validateAuditResults(items []immichapi.AssetBulkUploadCheckItem, results []immichapi.AssetBulkUploadCheckResult) string {
	if len(items) != len(results) {
		return "response_count_mismatch"
	}
	wanted := make(map[string]bool, len(items))
	for _, item := range items {
		wanted[item.Id] = true
	}
	for _, result := range results {
		if !wanted[result.Id] {
			return "response_unknown_or_duplicate_id"
		}
		delete(wanted, result.Id)
	}
	if len(wanted) != 0 {
		return "response_missing_id"
	}
	return ""
}

func classifyAuditResult(r immichapi.AssetBulkUploadCheckResult) (takeoutaudit.Status, string, string) {
	switch {
	case r.Action == immichapi.Accept && r.Reason == nil && r.AssetId == nil && r.IsTrashed == nil:
		return takeoutaudit.NotMatched, "no_byte_identical_record_in_account; external_libraries_not_covered", ""
	case r.Action == immichapi.Reject && r.Reason != nil && *r.Reason == immichapi.AssetRejectReasonDuplicate && r.AssetId != nil && *r.AssetId != uuid.Nil && r.IsTrashed != nil:
		if *r.IsTrashed {
			return takeoutaudit.Trashed, "matching_catalog_record_in_trash", r.AssetId.String()
		}
		return takeoutaudit.Match, "catalog_only; server_bytes_and_metadata_not_verified", r.AssetId.String()
	default:
		return takeoutaudit.Unknown, "unsupported_or_incomplete_result", ""
	}
}

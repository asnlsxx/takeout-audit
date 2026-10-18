# Takeout Audit

Explain what happened to every file in a Google Takeout → Immich migration.

Stream multiple ZIPs without extracting them, keep every source occurrence,
identify album duplicates, and compare each distinct media file with your
Immich account. The dedicated binary has no upload, deletion or self-update
commands.

**Alpha:** local/mock tests, native Windows/Linux/macOS checks and a disposable
Immich 3.2.0 lifecycle [passed in CI](https://github.com/asnlsxx/takeout-audit/actions/runs/39115445636).
Online checks accept stable Immich 3.2.x; other patches and real user exports
have not been individually validated. This is not a backup certificate or
permission to remove originals.

[中文快速开始](START-HERE.zh-CN.md) · [Technical guide](docs/takeout-audit.md) ·
[Validation](VALIDATION.zh-CN.md) · [Contributing](CONTRIBUTING.md)

## Try it

Download a [prerelease archive](https://github.com/asnlsxx/takeout-audit/releases)
for your platform, extract it and run:

```sh
takeout-audit --inventory-only --out inventory takeout-001.zip takeout-002.zip
```

Offline mode needs no config, account or network. It intentionally exits **2**:
remote media checks have not happened. Supply every ZIP from the same export;
the tool cannot know if you omitted an export part.

For online checks, copy `config.example.yaml` to `config.audit.yaml`, enter your
server origin (without `/api`) and an API key with `user.read` + `asset.upload`.
The checksum endpoint requires that permission despite performing no upload.

```sh
takeout-audit --config config.audit.yaml --out audit takeout-001.zip takeout-002.zip
```

The output directory must be new, its parent must exist, and it must be outside
input directories. An extracted directory can replace the ZIP arguments.
PowerShell users: pass a file array as shown in the Chinese quick start.

## What you get

| File | Purpose |
|---|---|
| `summary.json` | Source list, enumeration completeness, occurrence and distinct-content counts, errors, account/version scope |
| `ledger.jsonl` | Every observed file, including JSON, unsupported extensions and failures |
| `media.csv` | Filterable media rows with content group, source copies, status and reason |

A photo in three albums stays three source rows but is queried once. A matched
JPEG does not cover a missing MOV resource. Unknown responses, missing trash
flags and corrupt ZIP members never turn into a successful match.

| Exit | Meaning |
|---|---|
| 0 | Scoped content audit completed without attention items |
| 1 | Completed with differences, excluded/unclassified files or no media |
| 2 | Incomplete: offline mode, read/API/version errors or cancellation |

`catalog_match` is a server catalog checksum match, not a fresh check of server
file bytes. `not_matched` does not mean lost: edits, re-encoding, another account
and external libraries can explain it. Metadata, albums and export completeness
are explicitly outside scope. Never delete originals based only on this report.

## Why this exists

Migration users already use dry runs and debug logs, yet still need to explain
count differences. One user eventually [wrote a hash-grouping script to explain
album copies](https://www.reddit.com/r/immich/comments/1pvhgmk/dealing_with_discarded_assets_when_importing_from/).
Another [reported 43,000 source items versus roughly 34,000 processed items](https://www.reddit.com/r/immich/comments/1nh869r/immichgo_wont_upload_all_photos/).
These are reconciliation requests, not proof that Immich loses files.

[immich-admin-cli](https://github.com/dhcgn/immich-admin-cli) already checks local
file hashes; [PhotoProof](https://github.com/cmer/photoproof) addresses Apple Photos.
This fork adds a Takeout source ledger across ZIPs, duplicate-copy explanations,
and strict unknown/error accounting. It reuses the upstream foundation and
retains its MIT attribution. See [problem evidence and stop conditions](docs/problem-and-scope.md)
and [provenance](NOTICE.md).

## Build and verify

```sh
go build -o takeout-audit ./cmd/takeout-audit
go test ./...
go vet ./...
go generate ./...
```

Requires the Go version in `go.mod`. The inherited `immich-admin` binary still
supports `cw audit-takeout`, but prerelease packages ship only the dedicated
binary. Disposable real-server CI does not use private photos, production
credentials or ML model downloads.

<details>
<summary>Inherited upstream API coverage (generated)</summary>

<!-- API-TABLE:BEGIN -->
**31 of 267 endpoints implemented** (10 deprecated and 2 internal endpoints omitted per project policy).

<details>
<summary><b>API keys</b> (0/7)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/api-keys` | `getApiKeys` | Stable |
|  | POST | `/api-keys` | `createApiKey` | Stable |
|  | GET | `/api-keys/me` | `getMyApiKey` | Stable |
|  | DELETE | `/api-keys/{id}` | `deleteApiKey` | Stable |
|  | GET | `/api-keys/{id}` | `getApiKey` | Stable |
|  | PUT | `/api-keys/{id}` | `updateApiKey` | Stable |
|  | POST | `/api-keys/{id}/rotate` | `rotateApiKey` | – |

</details>

<details>
<summary><b>Activities</b> (0/4)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/activities` | `getActivities` | Stable |
|  | POST | `/activities` | `createActivity` | Stable |
|  | GET | `/activities/statistics` | `getActivityStatistics` | Stable |
|  | DELETE | `/activities/{id}` | `deleteActivity` | Stable |

</details>

<details>
<summary><b>Albums</b> (8/13)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
| ✅ | GET | `/albums` | `getAllAlbums` | Stable |
| ✅ | POST | `/albums` | `createAlbum` | Stable |
|  | PUT | `/albums/assets` | `addAssetsToAlbums` | Stable |
|  | GET | `/albums/statistics` | `getAlbumStatistics` | Stable |
| ✅ | DELETE | `/albums/{id}` | `deleteAlbum` | Stable |
| ✅ | GET | `/albums/{id}` | `getAlbumInfo` | Stable |
| ✅ | PATCH | `/albums/{id}` | `updateAlbumInfo` | Stable |
| ✅ | DELETE | `/albums/{id}/assets` | `removeAssetFromAlbum` | Stable |
| ✅ | PUT | `/albums/{id}/assets` | `addAssetsToAlbum` | Stable |
|  | GET | `/albums/{id}/map-markers` | `getAlbumMapMarkers` | – |
|  | DELETE | `/albums/{id}/user/{userId}` | `removeUserFromAlbum` | Stable |
|  | PUT | `/albums/{id}/user/{userId}` | `updateAlbumUser` | Stable |
| ✅ | PUT | `/albums/{id}/users` | `addUsersToAlbum` | Stable |

</details>

<details>
<summary><b>Asset files</b> (0/4)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/asset-files` | `searchAssetFiles` | Alpha |
|  | DELETE | `/asset-files/{id}` | `deleteAssetFile` | Alpha |
|  | GET | `/asset-files/{id}` | `getAssetFile` | Alpha |
|  | GET | `/asset-files/{id}/download` | `downloadAssetFile` | Alpha |

</details>

<details>
<summary><b>Assets</b> (8/26)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
| ✅ | DELETE | `/assets` | `deleteAssets` | Stable |
| ✅ | POST | `/assets` | `uploadAsset` | Stable |
|  | PUT | `/assets` | `updateAssets` | Stable |
| ✅ | POST | `/assets/bulk-upload-check` | `checkBulkUpload` | Stable |
| ✅ | PUT | `/assets/copy` | `copyAsset` | Stable |
|  | POST | `/assets/jobs` | `runAssetJobs` | Stable |
|  | DELETE | `/assets/metadata` | `deleteBulkAssetMetadata` | Beta |
|  | PUT | `/assets/metadata` | `updateBulkAssetMetadata` | Beta |
|  | GET | `/assets/statistics` | `getAssetStatistics` | Stable |
| ✅ | GET | `/assets/{id}` | `getAssetInfo` | Stable |
| ✅ | PUT | `/assets/{id}` | `updateAsset` | Stable |
|  | DELETE | `/assets/{id}/edits` | `removeAssetEdits` | Beta |
|  | GET | `/assets/{id}/edits` | `getAssetEdits` | Beta |
|  | PUT | `/assets/{id}/edits` | `editAsset` | Beta |
|  | GET | `/assets/{id}/metadata` | `getAssetMetadata` | Stable |
|  | PUT | `/assets/{id}/metadata` | `updateAssetMetadata` | Stable |
|  | DELETE | `/assets/{id}/metadata/{key}` | `deleteAssetMetadata` | Stable |
|  | GET | `/assets/{id}/metadata/{key}` | `getAssetMetadataByKey` | Stable |
|  | GET | `/assets/{id}/ocr` | `getAssetOcr` | Stable |
| ✅ | GET | `/assets/{id}/original` | `downloadAsset` | Stable |
| ✅ | GET | `/assets/{id}/thumbnail` | `viewAsset` | Stable |
|  | GET | `/assets/{id}/video/playback` | `playAssetVideo` | Stable |
|  | GET | `/assets/{id}/video/stream/main.m3u8` | `getMainPlaylist` | Alpha |
|  | DELETE | `/assets/{id}/video/stream/{sessionId}` | `endSession` | Alpha |
|  | GET | `/assets/{id}/video/stream/{sessionId}/{variantIndex}/playlist.m3u8` | `getMediaPlaylist` | Alpha |
|  | GET | `/assets/{id}/video/stream/{sessionId}/{variantIndex}/{filename}` | `getSegment` | Alpha |

</details>

<details>
<summary><b>Authentication</b> (0/17)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | POST | `/auth/admin-sign-up` | `signUpAdmin` | Stable |
|  | POST | `/auth/change-password` | `changePassword` | Stable |
|  | POST | `/auth/login` | `login` | Stable |
|  | POST | `/auth/logout` | `logout` | Stable |
|  | DELETE | `/auth/pin-code` | `resetPinCode` | Stable |
|  | POST | `/auth/pin-code` | `setupPinCode` | Stable |
|  | PUT | `/auth/pin-code` | `changePinCode` | Stable |
|  | POST | `/auth/session/lock` | `lockAuthSession` | Stable |
|  | POST | `/auth/session/unlock` | `unlockAuthSession` | Stable |
|  | GET | `/auth/status` | `getAuthStatus` | Stable |
|  | POST | `/auth/validateToken` | `validateAccessToken` | Stable |
|  | POST | `/oauth/authorize` | `startOAuth` | Stable |
|  | POST | `/oauth/backchannel-logout` | `logoutOAuth` | – |
|  | POST | `/oauth/callback` | `finishOAuth` | Stable |
|  | POST | `/oauth/link` | `linkOAuthAccount` | Stable |
|  | GET | `/oauth/mobile-redirect` | `redirectOAuthToMobile` | Stable |
|  | POST | `/oauth/unlink` | `unlinkOAuthAccount` | Stable |

</details>

<details>
<summary><b>Authentication (admin)</b> (0/1)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | POST | `/admin/auth/unlink-all` | `unlinkAllOAuthAccountsAdmin` | Stable |

</details>

<details>
<summary><b>Cluster groups</b> (0/8)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/cluster-groups/requests` | `getClusterGroupRequests` | – |
|  | DELETE | `/cluster-groups/requests/{id}` | `deleteClusterGroupRequest` | – |
|  | POST | `/cluster-groups/requests/{id}/accept` | `acceptClusterGroupRequest` | – |
|  | POST | `/cluster-groups/{id}/leave` | `leaveClusterGroup` | – |
|  | POST | `/cluster-groups/{id}/regenerate-people` | `clusterGroupRegeneratePeople` | – |
|  | GET | `/cluster-groups/{id}/requests` | `getClusterGroupRequestsForGroup` | – |
|  | PUT | `/cluster-groups/{id}/requests` | `createClusterGroupRequest` | – |
|  | GET | `/cluster-groups/{id}/users` | `getClusterGroupUsers` | – |

</details>

<details>
<summary><b>Config (admin)</b> (0/3)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/admin/config` | `getAdminConfig` | Alpha |
|  | PUT | `/admin/config` | `updateAdminConfig` | Alpha |
|  | GET | `/admin/config/defaults` | `getAdminConfigDefaults` | Alpha |

</details>

<details>
<summary><b>Config (public)</b> (0/2)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/public/config` | `getPublicConfig` | Alpha |
|  | GET | `/public/config/defaults` | `getPublicConfigDefaults` | Alpha |

</details>

<details>
<summary><b>Config (user)</b> (0/2)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/config` | `getUserConfig` | Alpha |
|  | GET | `/config/defaults` | `getUserConfigDefaults` | Alpha |

</details>

<details>
<summary><b>Database Backups (admin)</b> (0/5)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | DELETE | `/admin/database-backups` | `deleteDatabaseBackup` | Alpha |
|  | GET | `/admin/database-backups` | `listDatabaseBackups` | Alpha |
|  | POST | `/admin/database-backups/start-restore` | `startDatabaseRestoreFlow` | Alpha |
|  | POST | `/admin/database-backups/upload` | `uploadDatabaseBackup` | Alpha |
|  | GET | `/admin/database-backups/{filename}` | `downloadDatabaseBackup` | Alpha |

</details>

<details>
<summary><b>Download</b> (0/2)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | POST | `/download/archive` | `downloadArchive` | Stable |
|  | POST | `/download/info` | `getDownloadInfo` | Stable |

</details>

<details>
<summary><b>Duplicates</b> (0/4)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | DELETE | `/duplicates` | `deleteDuplicates` | Stable |
|  | GET | `/duplicates` | `getAssetDuplicates` | Stable |
|  | POST | `/duplicates/resolve` | `resolveDuplicates` | Alpha |
|  | DELETE | `/duplicates/{id}` | `deleteDuplicate` | Stable |

</details>

<details>
<summary><b>Faces</b> (0/4)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/faces` | `getFaces` | Stable |
|  | POST | `/faces` | `createFace` | Stable |
|  | DELETE | `/faces/{id}` | `deleteFace` | Stable |
|  | PUT | `/faces/{id}` | `reassignFacesById` | Stable |

</details>

<details>
<summary><b>Jobs</b> (0/1)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | POST | `/jobs` | `createJob` | Stable |

</details>

<details>
<summary><b>Libraries</b> (1/8)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/libraries` | `getAllLibraries` | Stable |
|  | POST | `/libraries` | `createLibrary` | Stable |
|  | DELETE | `/libraries/{id}` | `deleteLibrary` | Stable |
|  | GET | `/libraries/{id}` | `getLibrary` | Stable |
|  | PUT | `/libraries/{id}` | `updateLibrary` | Stable |
|  | POST | `/libraries/{id}/scan` | `scanLibrary` | Stable |
|  | GET | `/libraries/{id}/statistics` | `getLibraryStatistics` | Stable |
| ✅ | POST | `/libraries/{id}/validate` | `validate` | Stable |

</details>

<details>
<summary><b>Maintenance (admin)</b> (0/9)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/admin/integrity/report` | `getIntegrityReport` | Alpha |
|  | DELETE | `/admin/integrity/report/{id}` | `deleteIntegrityReport` | Alpha |
|  | GET | `/admin/integrity/report/{id}/file` | `getIntegrityReportFile` | Alpha |
|  | GET | `/admin/integrity/report/{type}/csv` | `getIntegrityReportCsv` | Alpha |
|  | GET | `/admin/integrity/summary` | `getIntegrityReportSummary` | Alpha |
|  | POST | `/admin/maintenance` | `setMaintenanceMode` | Alpha |
|  | GET | `/admin/maintenance/detect-install` | `detectPriorInstall` | Alpha |
|  | POST | `/admin/maintenance/login` | `maintenanceLogin` | Alpha |
|  | GET | `/admin/maintenance/status` | `getMaintenanceStatus` | Alpha |

</details>

<details>
<summary><b>Map</b> (0/2)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/map/markers` | `getMapMarkers` | Stable |
|  | GET | `/map/reverse-geocode` | `reverseGeocode` | Stable |

</details>

<details>
<summary><b>Memories</b> (0/8)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/memories` | `searchMemories` | Stable |
|  | POST | `/memories` | `createMemory` | Stable |
|  | GET | `/memories/statistics` | `memoriesStatistics` | Stable |
|  | DELETE | `/memories/{id}` | `deleteMemory` | Stable |
|  | GET | `/memories/{id}` | `getMemory` | Stable |
|  | PUT | `/memories/{id}` | `updateMemory` | Stable |
|  | DELETE | `/memories/{id}/assets` | `removeMemoryAssets` | Stable |
|  | PUT | `/memories/{id}/assets` | `addMemoryAssets` | Stable |

</details>

<details>
<summary><b>Notifications</b> (0/6)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | DELETE | `/notifications` | `deleteNotifications` | Stable |
|  | GET | `/notifications` | `getNotifications` | Stable |
|  | PUT | `/notifications` | `updateNotifications` | Stable |
|  | DELETE | `/notifications/{id}` | `deleteNotification` | Stable |
|  | GET | `/notifications/{id}` | `getNotification` | Stable |
|  | PUT | `/notifications/{id}` | `updateNotification` | Stable |

</details>

<details>
<summary><b>Notifications (admin)</b> (0/3)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | POST | `/admin/notifications` | `createNotification` | Stable |
|  | POST | `/admin/notifications/templates/{name}` | `getNotificationTemplateAdmin` | Stable |
|  | POST | `/admin/notifications/test-email` | `sendTestEmailAdmin` | Stable |

</details>

<details>
<summary><b>Partners</b> (0/4)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/partners` | `getPartners` | Stable |
|  | POST | `/partners` | `createPartner` | Stable |
|  | DELETE | `/partners/{id}` | `removePartner` | Stable |
|  | PUT | `/partners/{id}` | `updatePartner` | Stable |

</details>

<details>
<summary><b>People</b> (0/14)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | DELETE | `/people` | `deletePeople` | Stable |
|  | GET | `/people` | `getAllPeople` | Stable |
|  | POST | `/people` | `createPerson` | Stable |
|  | PUT | `/people` | `updatePeople` | Stable |
|  | POST | `/people/merge` | `mergePeople` | Stable |
|  | DELETE | `/people/users` | `removeUsersFromPeople` | Beta |
|  | GET | `/people/users` | `getUsersForPeople` | Beta |
|  | PUT | `/people/users` | `upsertPeopleUsers` | Beta |
|  | DELETE | `/people/{id}` | `deletePerson` | Stable |
|  | GET | `/people/{id}` | `getPerson` | Stable |
|  | PUT | `/people/{id}` | `updatePerson` | Stable |
|  | PUT | `/people/{id}/reassign` | `reassignFaces` | Stable |
|  | GET | `/people/{id}/statistics` | `getPersonStatistics` | Stable |
|  | GET | `/people/{id}/thumbnail` | `getPersonThumbnail` | Stable |

</details>

<details>
<summary><b>Plugins</b> (0/4)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/plugins` | `searchPlugins` | – |
|  | GET | `/plugins/methods` | `searchPluginMethods` | – |
|  | GET | `/plugins/templates` | `searchPluginTemplates` | – |
|  | GET | `/plugins/{id}` | `getPlugin` | – |

</details>

<details>
<summary><b>Queues</b> (0/5)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/queues` | `getQueues` | Alpha |
|  | GET | `/queues/{name}` | `getQueue` | Alpha |
|  | PUT | `/queues/{name}` | `updateQueue` | Alpha |
|  | DELETE | `/queues/{name}/jobs` | `emptyQueue` | Alpha |
|  | GET | `/queues/{name}/jobs` | `getQueueJobs` | Alpha |

</details>

<details>
<summary><b>Search</b> (1/9)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/search/cities` | `getAssetsByCity` | Stable |
|  | GET | `/search/explore` | `getExploreData` | Stable |
| ✅ | POST | `/search/metadata` | `searchAssets` | Stable |
|  | GET | `/search/person` | `searchPerson` | Stable |
|  | GET | `/search/places` | `searchPlaces` | Stable |
|  | POST | `/search/random` | `searchRandom` | Stable |
|  | POST | `/search/smart` | `searchSmart` | Stable |
|  | POST | `/search/statistics` | `searchAssetStatistics` | Stable |
|  | GET | `/search/suggestions` | `getSearchSuggestions` | Stable |

</details>

<details>
<summary><b>Server</b> (1/12)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/server/about` | `getAboutInfo` | Stable |
|  | GET | `/server/apk-links` | `getApkLinks` | Stable |
|  | DELETE | `/server/license` | `deleteServerLicense` | Stable |
|  | GET | `/server/license` | `getServerLicense` | Stable |
|  | PUT | `/server/license` | `setServerLicense` | Stable |
|  | GET | `/server/media-types` | `getSupportedMediaTypes` | Stable |
|  | GET | `/server/ping` | `pingServer` | Stable |
|  | GET | `/server/statistics` | `getServerStatistics` | Stable |
|  | GET | `/server/storage` | `getStorage` | Stable |
| ✅ | GET | `/server/version` | `getServerVersion` | Stable |
|  | GET | `/server/version-check` | `getVersionCheck` | Stable |
|  | GET | `/server/version-history` | `getVersionHistory` | Stable |

</details>

<details>
<summary><b>Sessions</b> (0/6)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | DELETE | `/sessions` | `deleteAllSessions` | Stable |
|  | GET | `/sessions` | `getSessions` | Stable |
|  | POST | `/sessions` | `createSession` | Stable |
|  | DELETE | `/sessions/{id}` | `deleteSession` | Stable |
|  | PUT | `/sessions/{id}` | `updateSession` | Stable |
|  | POST | `/sessions/{id}/lock` | `lockSession` | Stable |

</details>

<details>
<summary><b>Shared links</b> (0/9)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/shared-links` | `getAllSharedLinks` | Stable |
|  | POST | `/shared-links` | `createSharedLink` | Stable |
|  | POST | `/shared-links/login` | `sharedLinkLogin` | Beta |
|  | GET | `/shared-links/me` | `getMySharedLink` | Stable |
|  | DELETE | `/shared-links/{id}` | `removeSharedLink` | Stable |
|  | GET | `/shared-links/{id}` | `getSharedLinkById` | Stable |
|  | PATCH | `/shared-links/{id}` | `updateSharedLink` | Stable |
|  | DELETE | `/shared-links/{id}/assets` | `removeSharedLinkAssets` | Stable |
|  | PUT | `/shared-links/{id}/assets` | `addSharedLinkAssets` | Stable |

</details>

<details>
<summary><b>Stacks</b> (0/7)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | DELETE | `/stacks` | `deleteStacks` | Stable |
|  | GET | `/stacks` | `searchStacks` | Stable |
|  | POST | `/stacks` | `createStack` | Stable |
|  | DELETE | `/stacks/{id}` | `deleteStack` | Stable |
|  | GET | `/stacks/{id}` | `getStack` | Stable |
|  | PUT | `/stacks/{id}` | `updateStack` | Stable |
|  | DELETE | `/stacks/{id}/assets/{assetId}` | `removeAssetFromStack` | Stable |

</details>

<details>
<summary><b>Sync</b> (0/4)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | DELETE | `/sync/ack` | `deleteSyncAck` | Stable |
|  | GET | `/sync/ack` | `getSyncAck` | Stable |
|  | POST | `/sync/ack` | `sendSyncAck` | Stable |
|  | POST | `/sync/stream` | `getSyncStream` | Stable |

</details>

<details>
<summary><b>System config</b> (0/1)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/system-config/storage-template-options` | `getStorageTemplateOptions` | Stable |

</details>

<details>
<summary><b>System metadata</b> (0/4)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/system-metadata/admin-onboarding` | `getAdminOnboarding` | Stable |
|  | POST | `/system-metadata/admin-onboarding` | `updateAdminOnboarding` | Stable |
|  | GET | `/system-metadata/reverse-geocoding-state` | `getReverseGeocodingState` | Stable |
|  | GET | `/system-metadata/version-check-state` | `getVersionCheckState` | Stable |

</details>

<details>
<summary><b>Tags</b> (6/9)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
| ✅ | GET | `/tags` | `getAllTags` | Stable |
|  | POST | `/tags` | `createTag` | Stable |
| ✅ | PUT | `/tags` | `upsertTags` | Stable |
| ✅ | PUT | `/tags/assets` | `bulkTagAssets` | Stable |
| ✅ | DELETE | `/tags/{id}` | `deleteTag` | Stable |
| ✅ | GET | `/tags/{id}` | `getTagById` | Stable |
|  | PUT | `/tags/{id}` | `updateTag` | Stable |
|  | DELETE | `/tags/{id}/assets` | `untagAssets` | Stable |
| ✅ | PUT | `/tags/{id}/assets` | `tagAssets` | Stable |

</details>

<details>
<summary><b>Trash</b> (0/3)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | POST | `/trash/empty` | `emptyTrash` | Stable |
|  | POST | `/trash/restore` | `restoreTrash` | Stable |
|  | POST | `/trash/restore/assets` | `restoreAssets` | Stable |

</details>

<details>
<summary><b>Users</b> (3/16)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
| ✅ | GET | `/users` | `searchUsers` | Stable |
| ✅ | GET | `/users/me` | `getMyUser` | Stable |
|  | PUT | `/users/me` | `updateMyUser` | Stable |
|  | GET | `/users/me/calendar-heatmap` | `getMyCalendarHeatmap` | Stable |
|  | DELETE | `/users/me/license` | `deleteUserLicense` | Stable |
|  | GET | `/users/me/license` | `getUserLicense` | Stable |
|  | PUT | `/users/me/license` | `setUserLicense` | Stable |
|  | DELETE | `/users/me/onboarding` | `deleteUserOnboarding` | Stable |
|  | GET | `/users/me/onboarding` | `getUserOnboarding` | Stable |
|  | PUT | `/users/me/onboarding` | `setUserOnboarding` | Stable |
|  | GET | `/users/me/preferences` | `getMyPreferences` | Stable |
|  | PUT | `/users/me/preferences` | `updateMyPreferences` | Stable |
|  | DELETE | `/users/profile-image` | `deleteProfileImage` | Stable |
|  | POST | `/users/profile-image` | `createProfileImage` | Stable |
| ✅ | GET | `/users/{id}` | `getUser` | Stable |
|  | GET | `/users/{id}/profile-image` | `getProfileImage` | Stable |

</details>

<details>
<summary><b>Users (admin)</b> (0/11)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/admin/users` | `searchUsersAdmin` | Stable |
|  | POST | `/admin/users` | `createUserAdmin` | Stable |
|  | DELETE | `/admin/users/{id}` | `deleteUserAdmin` | Stable |
|  | GET | `/admin/users/{id}` | `getUserAdmin` | Stable |
|  | PUT | `/admin/users/{id}` | `updateUserAdmin` | Stable |
|  | GET | `/admin/users/{id}/calendar-heatmap` | `getUserCalendarHeatmapAdmin` | Stable |
|  | GET | `/admin/users/{id}/preferences` | `getUserPreferencesAdmin` | Stable |
|  | PUT | `/admin/users/{id}/preferences` | `updateUserPreferencesAdmin` | Stable |
|  | POST | `/admin/users/{id}/restore` | `restoreUserAdmin` | Stable |
|  | GET | `/admin/users/{id}/sessions` | `getUserSessionsAdmin` | Stable |
|  | GET | `/admin/users/{id}/statistics` | `getUserStatisticsAdmin` | Stable |

</details>

<details>
<summary><b>Views</b> (0/2)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
|  | GET | `/view/folder` | `getAssetsByOriginalPath` | Stable |
|  | GET | `/view/folder/unique-paths` | `getUniqueOriginalPaths` | Stable |

</details>

<details>
<summary><b>Workflows</b> (3/8)</summary>

| Impl | Method | Path | Operation | State |
|:----:|--------|------|-----------|-------|
| ✅ | GET | `/workflows` | `searchWorkflows` | – |
|  | POST | `/workflows` | `createWorkflow` | – |
|  | GET | `/workflows/triggers` | `getWorkflowTriggers` | – |
|  | DELETE | `/workflows/{id}` | `deleteWorkflow` | – |
| ✅ | GET | `/workflows/{id}` | `getWorkflow` | – |
|  | PUT | `/workflows/{id}` | `updateWorkflow` | – |
| ✅ | GET | `/workflows/{id}/logs` | `getWorkflowLogs` | – |
|  | GET | `/workflows/{id}/share` | `getWorkflowForShare` | – |

</details>
<!-- API-TABLE:END -->

</details>

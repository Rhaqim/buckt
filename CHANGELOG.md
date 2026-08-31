# Changelog

All notable changes to buckt are documented here. This project follows
[Semantic Versioning](https://semver.org/).

## [1.10.0] — unreleased

A backward-compatible **minor** release adding file expiry / temp files.
Additive; no API removals.

### ✨ Added

- **Expiring / temp files** (`SetFileTTL`, `SetFileExpiry`, `PurgeExpired`,
  `WithExpirySweeper`). Give a file a TTL and buckt permanently deletes it —
  blob, image derivatives, and metadata row — once it's due, emitting a
  `file.purged` event. A new indexed `expires_at` column (schema migration v6,
  additive) makes the sweep a cheap ranged query. The sweep is app-driven by
  default (call `PurgeExpired` from your own scheduler); `WithExpirySweeper`
  opts into a built-in background ticker (stopped by `Client.Close`). Built to
  extend toward general scheduled actions (e.g. "email this after 24h") on the
  same event-backed sweep.

## [1.9.0] — 2026-08-11

A backward-compatible **minor** release adding a pluggable upload-scanning hook.
Additive; no API removals.

### ✨ Added

- **Pluggable upload scanning** (`WithUploadScanner`, `pkg/scan`,
  `ErrUploadRejected`). Register a `scan.Scanner` and buckt runs it on every
  upload's bytes at the single write chokepoint, before anything is committed —
  return an error to reject the file (malware, disallowed type) and nothing is
  stored, no `file.uploaded` event fires, and the caller gets `ErrUploadRejected`
  wrapping the scanner's reason (branch with `errors.Is`). buckt ships no
  scanning engine by design (no AV dependencies in a storage library); the
  application supplies one — a ClamAV client, a VirusTotal lookup, a
  content-type allowlist, etc. Unlike event handlers (which fire *after* commit),
  the scanner runs *before* it, so it can actually block. Nil by default — zero
  overhead when unused.
- **Repo secret-scanning** — a gitleaks pre-commit hook
  (`.pre-commit-config.yaml` + `.gitleaks.toml`) and a `secret-scan` GitHub
  Actions workflow catch hardcoded credentials before they reach history.

### ⚡ Changed

- **`MigrateAll` now copies files concurrently** with a bounded worker pool
  instead of one at a time, and checks the target with a single `List` up front
  rather than an `Exists` round-trip per file. On a high-latency target (S3/R2)
  this is dramatically faster for large buckets. Concurrency is configurable via
  `MigrationConfig.Concurrency` (default `DefaultMigrationConcurrency` = 8);
  higher values trade memory (each in-flight file is buffered) and provider
  rate-limit headroom for throughput. Still idempotent and safe to re-run after
  an interruption. Individual file copies now retry transient failures with a
  linear backoff, so a flaky target no longer silently drops files.
- **`MigrationStatus` progress now always reaches `total`.** A file that
  permanently fails after retries is counted as processed (not dropped), so
  `completed` reaches `total` when the run finishes and a progress badge no
  longer hangs. The new `Client.MigrationFailures(ctx)` reports how many objects
  could not be copied (re-run `MigrateAll` to retry them); the web `/backend`
  endpoint gained a `failed` field.
- **`MigrateAll` is now resumable.** Each copied object is recorded in the
  database (`buckt_migration_models`), so a migration interrupted by a
  restart/crash resumes from where it left off — already-copied files are skipped
  straight from the persisted state without re-scanning the target, and progress
  no longer restarts from zero. Persistence is best-effort (a recording failure
  never fails an idempotent copy) and keyed by the target backend's name.

### 🧹 Removed

- Deleted the unused `internal/migration` package (a second, never-wired
  migration service). Its worthwhile piece — per-file retry with backoff — was
  folded into the live migration path above.

## [1.8.0] — 2026-08-11

A backward-compatible **minor** release on top of v1.7.0 adding controls to
drive and observe a local → cloud storage migration. Additive; no API removals.

### ✨ Added

- **Bulk migration controls** (`Client.MigrateAll`, `Client.MigrationStatus`,
  `Client.BackendName`). When a Client is created with `WithMigration` (local →
  cloud dual-write), these drive and observe a background copy of the files that
  predate the cutover — `MigrateAll` schedules the copy (skipping objects already
  in the target, so it is idempotent and safe to re-run after an interruption),
  `MigrationStatus` reports `completed`/`total`, and
  `BackendName` returns the active backend (`local`, `s3`, or `local->s3` mid
  migration). `MigrateAll`/`MigrationStatus` return `ErrBackendUnavailable` /
  `ok=false` when migration isn't enabled. The bundled web UI adds a header badge
  (fed by a new `GET /backend` endpoint) showing which backend is in use and, in
  migration mode, both backends plus live copy progress. The `ui` example now
  takes `-mode=local|migrate|r2` (same features across modes, shared
  `db.sqlite` + `media/`) — `make ui MODE=migrate` walks the local → R2 →
  direct-R2 lifecycle; `example/migration/headless` is the non-UI equivalent.

### 🐛 Fixed

- **Files bulk-migrated to S3/R2 were unreachable (`NoSuchKey` on read).** In
  nested-namespace mode buckt keys blobs by a path with a leading slash
  (`/user/…`), but `MigrateAll` copies objects under the keys the local backend's
  `List()` reports, which are relative and drop the leading slash (`user/…`). The
  S3 backend used keys verbatim, so migrated objects landed under a key reads
  never asked for — directly-uploaded files worked, bulk-migrated ones 404'd. The
  S3 backend now transparently handles both key forms: reads (`Get`/`Stream`)
  fall back to the slash-stripped key, `Exists` checks both, and `Delete` /
  `DeleteFolder` / `Move` operate on both so nothing is orphaned. Writes are
  unchanged, so existing deployments keep working with no re-keying. The same
  fix is applied to the Azure Blob and Google Cloud Storage backends, which
  carried the identical latent mismatch.

## [1.7.0]

A backward-compatible **minor** release on top of v1.6.0 that adds opt-in
primitives making buckt a better fit for event-driven and media-heavy apps —
file lifecycle events, arbitrary file metadata, content deduplication, and
image derivatives (with a pluggable, WebP-capable processor). All additive; no
API removals, safe to adopt.

### ✨ Added

- **File lifecycle events** (`WithEventHandler`, new `pkg/events`). Opt-in,
  dependency-free. Register one or more handlers and buckt calls them after each
  successful file operation with an `events.Event` (`file.uploaded`,
  `file.trashed`, `file.restored`, `file.purged`) carrying storage-level facts
  (user, file id, name, path, content type, size, content hash). buckt performs
  no processing itself — this is the seam for enqueuing uploads to an AI/OCR
  worker, generating derivatives, cache warming, etc., keeping the dependency
  one-way (your worker imports buckt, never the reverse). Handlers run
  synchronously after the operation (outside the DB transaction) and are isolated
  by panic recovery, so a slow or panicking handler can't corrupt or fail the
  originating call. No handlers = zero overhead.
- **Arbitrary file metadata** (`SetFileMetadata` / `GetFileMetadata`). Attach a
  `map[string]string` of application data to any file — resource links (e.g. an
  order id, so an app can model attachments without buckt knowing about orders),
  AI-extracted tags, OCR text, etc. Stored as JSON on the file row via GORM's
  built-in serializer (no new dependency), added by migration `v4` (additive,
  non-destructive), and returned on `GetFile`. buckt never interprets it.
- **Content deduplication** (`WithDedup`). Opt-in. When enabled, an upload whose
  bytes hash-match a file already in the same target folder (for the same owner)
  returns that existing file's ID and skips writing the blob again — collapsing
  the "same photo sent four times" case into one stored object and saving storage
  and bandwidth. It reuses the content hash buckt already records, needs no schema
  change, is scoped to the target folder so it composes with nested-namespace
  paths, and never resurrects a trashed duplicate. Off by default.
- **Image derivatives** (`WithImageDerivatives`, `GenerateDerivatives`,
  `GetDerivative`). Opt-in. Configure named resized variants (e.g. `thumbnail`
  200px, `medium` 800px); `GenerateDerivatives(fileID)` produces and stores them
  for image uploads, and `GetDerivative(fileID, name)` fetches one — so the
  dashboard serves a thumbnail instead of the full-size original. Resizing is
  CPU-bound, so call `GenerateDerivatives` from a `file.uploaded` event handler
  (your worker), keeping it off the upload path. The built-in resizer is pure-Go
  JPEG/PNG (no new core dependency). Variants are stored as backend objects keyed
  by file id — moving or trashing the file never orphans them — never upscale,
  are removed on permanent delete, and their bytes are counted in
  `Client.StorageBytes` (new `derivatives_bytes` column, migration `v5`). The
  bundled web UI serves them via `GET /serve/:file_id/derivative/:name` (falling
  back to the original when a variant is absent): the file grid shows the
  `thumbnail` variant, clicking an image opens a preview using the larger
  `medium` variant with a link to the full-size original, and a "Regenerate
  previews" action (`POST /web/regenerate-derivatives/:file_id`) rebuilds them
  on demand.
- **Pluggable image processor** (`WithImageProcessor`, new `pkg/imageproc`).
  Derivative encoding goes through an `imageproc.Processor` interface, so you can
  add formats like **WebP** by supplying a processor from a module that imports
  the encoder — buckt's core stays free of it (the same one-way rule the cloud
  backends follow; a processor implements a stdlib-only signature and never
  imports buckt). Ships with a ready-made **pure-Go WebP** processor in the new
  `github.com/Rhaqim/buckt/imageproc/webp` module (`webp.New()`, no cgo); set a
  `DerivativeSpec.Format` of `"webp"` to use it.

## [1.6.0]

A backward-compatible **minor** release on top of v1.5.0: built-in backend
metrics, a bundled web UI refresh (dark mode, trash browsing, restore), plus
S3/Azure/GCP fixes. No API removals; safe to adopt.

> **If you use the Azure or GCP backend, upgrade.** v1.5.0's `Exists` is broken
> on both — the Azure backend **panics** on any blob-properties error (including
> the routine "not found" case), and the GCP backend can misreport a missing
> object. Both are fixed here (see **Fixed**).

### ✨ Added

- **Built-in backend metrics** (`WithMetrics`, new `pkg/metrics`). Opt-in,
  dependency-free (stdlib only). A metering decorator wraps every backend (local,
  S3, Azure, GCP, and the migration source/target) and records one operation per
  call — `put/get/list/exists/move/delete/delete_folder` — with counts, bytes,
  errors, and duration, into a `metrics.Collector` you read via `Snapshot()`. The
  S3 backend additionally reports exact per-API-call ops (`s3:PutObject`,
  `s3:GetObject`, `s3:ListObjectsV2`, …, capturing pagination) so Cloudflare R2
  Class A/B operations can be counted precisely; `metrics.R2Class` maps op names
  to R2 billing classes. New `Client.StorageBytes(ctx)` (total stored bytes = R2
  storage dimension) and `Client.CacheStats()` (cache hits/misses = backend reads
  avoided). Nil metrics (the default) means zero overhead. The backend↔buckt hook
  uses a stdlib-only callback signature, so cloud backends emit metrics without
  importing buckt (the dependency stays one-way).
- **`Endpoint` override for the Azure and GCP backends** (`azure.Config.Endpoint`,
  `gcp.Config.Endpoint`), matching the AWS backend. Enables emulators (Azurite,
  fake-gcs-server) and private/self-hosted deployments. For GCP an endpoint skips
  credential-file auth.
- **`Client.MetricsSnapshot()`** exposes the collected metrics from the `Client`
  itself (when configured with a `metrics.Collector`), and the bundled web client
  gained a **`GET /metrics`** JSON endpoint (storage bytes, cache hit/miss, and
  per-backend op counts with R2 class). The web examples now enable metrics, so
  running `example/client/web` and hitting `/metrics` shows live numbers.
- **Bundled web UI refresh** (`client/web` templates): **dark mode** (system-aware
  with a toggle), a **responsive** mobile-friendly layout, a **grid/list** view
  toggle, a breadcrumb, and a **"Usage" panel** that renders the metrics (storage,
  cache hit rate, and per-op R2 A/B/free counts) right in the browser.
- **Trash browsing in the bundled web UI.** A new **Trash** view (`GET /web/trash`)
  lists items that were moved to trash, with **Restore** and **Delete permanently**
  actions. Previously trashed items were only reachable via the
  `Client.GetTrashFolder` API.
- **Restore returns items to their original location.** Trashing a file or folder
  now records where it came from (new nullable `origin_parent_id` column, added by
  migration `v3` — additive and non-destructive), and `Client.RestoreFile` /
  `Client.RestoreFolder` (and the web Restore buttons) move it back there instead
  of dumping it in root. If the original folder no longer exists (or was itself
  trashed), restore falls back to root. Items trashed before this release have no
  recorded origin and restore to root. In nested-namespace mode the underlying
  blobs are physically moved back, inside the same transaction, so a backend
  failure rolls the restore back.

### 🐛 Fixed

- **S3-compatible endpoints now use `BaseEndpoint` + path-style** instead of a
  custom endpoint resolver that dropped the bucket from the request. Path-style
  stores (e.g. MinIO) previously failed with `NoSuchBucket`; they now work.
  Cloudflare R2 is unaffected (it already worked and continues to).
- **Azure `Exists` no longer panics.** It called `errors.As` with a
  `bloberror.Code` (a string), which panics on any `GetProperties` error —
  including the common not-found case. Now uses `bloberror.HasCode`.
- **GCP `Exists` correctly detects missing objects.** It compared the error with
  `==` against `storage.ErrObjectNotExist`, which fails when the error is wrapped
  (as GCS returns it). Now uses `errors.Is`.
- **Bundled web UI: viewing a file no longer returns `401`.** v1.5.0 moved
  `/serve` and `/stream` behind the header-based API guard, but browsers can't
  attach the `buckt-User-ID` header to `<img>`, `<video>`, or download requests,
  so the standalone UI (`WebModeUI`/`WebModeAll`) broke. These endpoints are now
  scoped by mode: the single-tenant UI serves content as the default owner (like
  the `/web` routes), standalone API mode still requires the header, and mount
  mode still defers to the mounting app's auth.
- **Usage panel distinguishes "metrics off" from "no ops yet."** `/metrics` now
  reports `metrics_enabled` from whether `WithMetrics` was configured, not from
  whether any backend op has been recorded, so a freshly started app with metrics
  enabled no longer claims metrics are off. The mount example now enables metrics.
- **Moving a non-empty folder to trash no longer fails with a duplicate-key
  error.** `DeleteFolder` updated the folder through a struct that had its child
  files/folders preloaded, so GORM tried to re-insert those children. It now
  updates by a bare model keyed on id (as the descendant path rewrites already
  did). This affected non-empty folders; empty folders were unaffected.

### 🧪 Tests

- Config-validation unit tests and emulator-backed contract tests (Put/Get/
  Exists/List/Move/Delete/DeleteFolder) for all three cloud backends — AWS via
  MinIO, Azure via Azurite, GCP via fake-gcs-server. Integration tests are gated
  by env vars and skip when unset.

## [1.5.0]

A **backward-compatible minor** release. The public API stays source-compatible
with v1.4.1, and the database is upgraded in place, non-destructively, the first
time you run it. **Take a database backup before upgrading a production system**
anyway — the upgrade rewrites schema and relocates trashed rows.

### Notable

- **`go` toolchain requirement raised to 1.26** (from 1.24) across all modules.
  This is not an API break, but consumers must build with Go 1.26+.
- **Soft-delete is replaced by a real `__trash__` folder.** The `DeletedAt` field
  on `FileModel` / `FolderModel` is **kept for source compatibility** (code
  reading `.DeletedAt` still compiles) but is now deprecated: it is ignored by
  the database and never populated (always `Valid == false`). Enumerate trashed
  items via `Client.GetTrashFolder` instead of inspecting `DeletedAt`.

### 💾 Database upgrade (automatic, non-destructive)

The first `buckt.New(...)` on a v1.4.1 database runs a **versioned, ledger-tracked
migration runner** (new `internal/database/schema` package, modelled on the
[loom](https://github.com/Rhaqim/loom) engine's `schema` package). It is safe to
call on every startup — applied migrations are recorded in a
`buckt_schema_migrations` ledger and never re-run.

- **v1.4.1 trashed data is preserved, not purged.** v1.4.1 hid deleted items with
  a `deleted_at` timestamp. Earlier development builds of this branch
  *hard-deleted* those rows on upgrade — that has been replaced. Trashed files and
  folders are now **relocated into each owner's `__trash__` folder** (keeping
  their storage path so blobs stay readable in nested-namespace mode), and only
  then is the `deleted_at` column dropped. **No data is lost.**
- Each migration runs in a transaction together with its ledger insert, so a
  failed upgrade rolls back cleanly and never records a partial version.

### ✨ Added

- **Typed sentinel errors** for programmatic error handling. `Client` methods now
  wrap failures with `errors.Is`-matchable sentinels — `ErrNotFound`,
  `ErrInvalidID`, `ErrInvalidName`, `ErrAlreadyExists`, `ErrFileTooLarge`,
  `ErrPathTraversal`, `ErrTrashBatchExceeded`, `ErrBackendUnavailable` — so
  callers can map failures to HTTP status codes without importing GORM. Available
  as `buckt.ErrNotFound` and, identically, `buckterr.ErrNotFound` (new
  `pkg/buckterr` package). The "record not found" case no longer leaks
  `gorm.ErrRecordNotFound`. The optional `LogConfig` logger is now purely for
  diagnostics — error handling should use these returned errors.
- **Errors carry context and are no longer logged by the library.** Internally,
  errors are now wrapped with `fmt.Errorf(...: %w)` so the returned error carries
  full context (previously the context was only written to the log and the raw
  error returned). buckt no longer logs the errors it returns; the caller
  decides whether to log. Errors that buckt intentionally swallows (best-effort
  cache/secondary-backend operations) are still surfaced via `Warn`.
- **`Client.Close()` now returns an error** (the database close error), matching
  the `io.Closer` idiom. Source-compatible: existing `defer client.Close()` and
  `client.Close()` statements keep compiling.
- **Configurable table prefix** (`DBConfig.TablePrefix` / `WithTablePrefix`). Set
  it for a **fresh** database that shares a schema with other tables; every buckt
  table and the migration ledger are prefixed. The default is empty, preserving
  the historical `folder_models` / `file_models` names so existing databases are
  adopted unchanged. buckt **refuses to start** if a prefix is set while legacy
  un-prefixed tables exist, rather than silently creating an empty prefixed schema
  and stranding your data.
- Cloudflare **R2 / S3-compatible** backends via `cloud/aws` (`Config.Endpoint`,
  `UsePathStyle`; R2 auto-detected) — see `example/cloud/cloudflare`. Combine with
  `RegisterPrimaryBackend` / `RegisterSecondaryBackend` + `WithMigration` for a
  live local→cloud cutover.

### 🔒 Security

- **File-name validation** (`utils.ValidateFileName`) is now enforced in
  `UploadFile`, `UpdateFile`, and `RenameFile`, matching the existing folder-name
  validation. Prevents path-traversal / cross-tenant object keys from crafted
  upload names (e.g. `../../other-user/...`). `UpdateFile` no longer builds paths
  by raw string concatenation.
- **Web serve hardening** (bundled `client/web`): non-media content types (HTML,
  SVG, …) are served as `attachment` instead of `inline`, plus a restrictive
  `Content-Security-Policy`, closing a stored-XSS vector. `/serve` and `/stream`
  are now behind the API guard (were previously unauthenticated). These changes
  affect only the optional bundled web client, not the core library or your own
  routing.
- **LIKE-metacharacter escaping** in folder move/rename/delete path rewrites, so a
  folder or file name containing `%` or `_` can no longer act as a wildcard and
  corrupt unrelated sibling paths.

### Upgrade guide (from v1.4.1)

1. **Back up your database** (and, in nested-namespace mode, your media directory).
2. Bump your Go toolchain to 1.26.
3. Rebuild against v1.5.0 — no import-path change and no source changes are
   required. If you referenced `.DeletedAt`, it still compiles, but migrate that
   logic to `GetTrashFolder` since the field is now always empty.
4. Start your app. The schema migration runs automatically and preserves trashed
   data. Verify your trashed items now appear under the `__trash__` folder.
5. Do **not** set `TablePrefix` on an existing database — it points buckt at a
   different, empty set of tables. Only use it for fresh databases (buckt refuses
   to start if you set one over legacy un-prefixed tables).

## [1.4.1]

Previous release. See the git history for details.

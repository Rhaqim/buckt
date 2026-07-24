# Changelog

All notable changes to buckt are documented here. This project follows
[Semantic Versioning](https://semver.org/).

## [1.5.0] — unreleased

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

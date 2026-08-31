# Buckt

> A flexible Go media storage library with pluggable cloud backends, image derivatives, lifecycle events, content dedup, upload scanning, built-in trash, dual-write migration, and an optional web UI.

[![Go Report Card](https://goreportcard.com/badge/github.com/Rhaqim/buckt)](https://goreportcard.com/report/github.com/Rhaqim/buckt)
[![GoDoc](https://godoc.org/github.com/Rhaqim/buckt?status.svg)](https://pkg.go.dev/github.com/Rhaqim/buckt)
[![License](https://img.shields.io/github/license/Rhaqim/buckt)](LICENCE)

Buckt is a media management library for Go applications that need to upload, organize, and manage files without reinventing media handling for every project.

Unlike object storage servers such as MinIO, Buckt focuses on the application layer. It provides a consistent API for managing media, folders, metadata, and storage backends, allowing your application to remain independent of where files are ultimately stored.

Out of the box, Buckt supports local filesystem storage with SQLite for development, and can seamlessly switch to Amazon S3, Google Cloud Storage, Azure Blob Storage, Cloudflare R2, or other backends with minimal configuration. Your application code remains unchanged while Buckt handles storage orchestration and media management.

Use Buckt when your application needs more than just object storage—when you need to organize media, manage file metadata, build folder hierarchies, and keep storage concerns separate from your business logic.

Why Buckt instead of MinIO?

Buckt and MinIO solve different problems.

MinIO is an object storage server. It stores and retrieves objects using the S3 API.

Buckt is a media management library. It sits inside your Go application and manages media workflows while using a storage backend such as the local filesystem, Amazon S3, Cloudflare R2, Azure Blob Storage or even MinIO itself.

Think of it this way:

* MinIO answers: “Where should the bytes be stored?”
* Buckt answers: “How should my application manage media?”

Buckt handles concerns such as:

* Uploading and serving files
* Folder organization
* File metadata
* Storage abstraction
* Backend portability
* Media lifecycle management

while delegating the actual storage of file data to your chosen backend.

In fact, Buckt can use MinIO as its storage backend, allowing you to combine MinIO’s object storage capabilities with Buckt’s higher-level media management features.

```sh
┌─────────────────────────────────────────────────────────────┐
│                      Your Application                       │
└──────────────────────────┬──────────────────────────────────┘
                           │
                  ┌────────▼─────────┐
                  │   buckt.Client   │  ◄── direct API or web client
                  └────────┬─────────┘
            ┌──────────────┼──────────────┐
            │              │              │
       ┌────▼────┐   ┌─────▼─────┐  ┌─────▼──────┐
       │ Folder  │   │   File    │  │   Cache    │
       │ Service │   │  Service  │  │  Manager   │
       └────┬────┘   └─────┬─────┘  └────────────┘
            │              │
       ┌────▼──────────────▼─────┐
       │   Repository (GORM)     │  ◄── SQLite or Postgres
       └────┬────────────────────┘
            │
       ┌────▼─────────────────────────────────────┐
       │            FileBackend                   │
       │  ┌──────┐ ┌─────┐ ┌─────┐ ┌──────┐ ┌──┐  │
       │  │Local │ │ S3  │ │ GCS │ │Azure │ │R2│  │
       │  └──────┘ └─────┘ └─────┘ └──────┘ └──┘  │
       └──────────────────────────────────────────┘
```

---

## Table of Contents

- [Buckt](#buckt)
  - [Table of Contents](#table-of-contents)
  - [Features](#features)
  - [Installation](#installation)
  - [Quick Start](#quick-start)
  - [Configuration](#configuration)
    - [All Options](#all-options)
  - [Storage Backends](#storage-backends)
    - [Local Filesystem](#local-filesystem)
    - [AWS S3](#aws-s3)
    - [Cloudflare R2](#cloudflare-r2)
    - [Google Cloud Storage](#google-cloud-storage)
    - [Azure Blob Storage](#azure-blob-storage)
  - [Migration Between Backends](#migration-between-backends)
  - [Image Derivatives](#image-derivatives)
  - [File Metadata](#file-metadata)
  - [Deduplication](#deduplication)
  - [Lifecycle Events](#lifecycle-events)
  - [Upload Scanning](#upload-scanning)
  - [Metrics](#metrics)
  - [Expiring / Temp Files](#expiring--temp-files)
  - [Trash \& Deletion](#trash--deletion)
  - [Web Client](#web-client)
    - [Modes](#modes)
    - [Features in the UI](#features-in-the-ui)
  - [Database](#database)
    - [Schema](#schema)
  - [Caching](#caching)
  - [Security](#security)
  - [API Reference](#api-reference)
    - [Folder Operations](#folder-operations)
    - [File Operations](#file-operations)
  - [Examples](#examples)
  - [Development](#development)
  - [License](#license)

---

## Features

| Feature | Description |
|---|---|
| 📁 **Folder Hierarchy** | Logical folder tree with rename, move, and recursive operations |
| ☁️ **Pluggable Backends** | Local FS, S3, GCS, Azure Blob, Cloudflare R2 — all behind one interface |
| 🔄 **Live Migration** | Dual-write between two backends with lazy forward migration on read + bulk copy |
| 🖼️ **Image Derivatives** | Generate thumbnails/resized variants on upload; pluggable, WebP-capable processor |
| 🪝 **Lifecycle Events** | Hook into `file.uploaded` / `trashed` / `restored` / `purged` for async work |
| 🛡️ **Upload Scanning** | Pluggable pre-write hook to reject malware/disallowed files before they're stored |
| 🧬 **Content Dedup** | Collapse identical uploads in a folder to a single stored blob |
| 🏷️ **File Metadata** | Attach arbitrary key/value metadata to any file |
| ⏳ **Expiring / Temp Files** | Give a file a TTL; an app-driven or background sweep permanently deletes it when due |
| 📊 **Metrics** | Per-backend operation counters (count, errors, bytes, latency) via a pluggable recorder |
| 🗑️ **Trash & Restore** | Soft-delete moves items to a per-user trash folder, hard-delete is one click away |
| 🌐 **Web UI** | Optional Gin-based dashboard with drag-and-drop, breadcrumbs, and previews |
| 🔌 **HTTP API** | RESTful endpoints for upload, download, stream, list, move, delete |
| 🗄️ **BYO Database** | SQLite by default, plug in any `*sql.DB` (Postgres tested) |
| ⚡ **In-Memory Cache** | LRU file cache + pluggable cache manager interface |
| 🔒 **Security First** | Path traversal protection, content-type validation, file size limits, RFC 6266 headers |
| 📦 **Flat or Nested** | Choose flat-namespace storage (UUID filenames) or hierarchical (folder paths) |

---

## Installation

```bash
go get github.com/Rhaqim/buckt
```

For cloud backends and the web client, install only what you need (each is a separate Go module to keep the core lean):

```bash
go get github.com/Rhaqim/buckt/cloud/aws        # AWS S3 + Cloudflare R2
go get github.com/Rhaqim/buckt/cloud/gcp        # Google Cloud Storage
go get github.com/Rhaqim/buckt/cloud/azure      # Azure Blob Storage
go get github.com/Rhaqim/buckt/client/web       # Web UI + HTTP API
```

---

## Quick Start

```go
  package main

  import (
    "log"
    "github.com/Rhaqim/buckt"
  )

  func main() {
    client, err := buckt.Default()
    if err != nil {
      log.Fatal(err)
    }
    defer client.Close()

    // Upload
    fileID, err := client.UploadFile(
      "user123",       // user ID
      "",              // parent folder ID (empty = root)
      "hello.txt",     // file name
      "text/plain",    // content type
      []byte("hello world"),
    )
    if err != nil {
      log.Fatal(err)
    }

    log.Println("Uploaded:", fileID)

    // Read it back
    file, _ := client.GetFile(fileID)
    log.Println(string(file.Data))
  }
```

That's it. Files land in `./media`, metadata in `./db.sqlite`, no extra setup.

---

## Configuration

Buckt uses a functional-options pattern. All options are optional — `Default()` gives you sensible defaults.

```go
  client, err := buckt.New(buckt.Config{
    MediaDir:       "./media",
    FlatNameSpaces: false,
    MaxFileSize:    100 * 1024 * 1024, // 100MB
  })
```

Or with options:

```go
  client, err := buckt.Default(
    buckt.WithLog(buckt.LogConfig{LogTerminal: true, LogFile: "logs"}),
    buckt.WithDB(buckt.Postgres, sqlDB),
    buckt.WithBackend(s3Backend),
    buckt.WithMaxFileSize(buckt.DefaultMaxFileSize),
    buckt.MediaDir("./uploads"),
    buckt.FlatNameSpaces(true),
  )
```

### All Options

| Option | Description |
|---|---|
| `WithLog(LogConfig)` | Configure terminal/file logging or pass a custom `*log.Logger` |
| `WithDB(driver, *sql.DB)` | Bring your own database connection (Postgres or SQLite) |
| `WithTablePrefix(string)` | Prefix buckt's table names to share a database with other apps |
| `WithCache(CacheConfig)` | Plug in a custom cache manager + tune the LRU file cache |
| `WithBackend(Backend)` | Set a single storage backend |
| `WithMigration(MigrationConfig)` | Dual-write migration between two backends |
| `WithImageDerivatives(...DerivativeSpec)` | Define resized image variants (thumbnails, etc.) — see [Image Derivatives](#image-derivatives) |
| `WithImageProcessor(imageproc.Processor)` | Swap the image processor (e.g. add WebP) |
| `WithEventHandler(events.Handler)` | Register a post-operation lifecycle hook — see [Lifecycle Events](#lifecycle-events) |
| `WithUploadScanner(scan.Scanner)` | Reject uploads before they're stored — see [Upload Scanning](#upload-scanning) |
| `WithDedup()` | Collapse identical uploads in a folder to one blob — see [Deduplication](#deduplication) |
| `WithMetrics(metrics.Recorder)` | Record per-backend operation metrics — see [Metrics](#metrics) |
| `WithExpirySweeper(time.Duration)` | Run a background ticker that purges expired files — see [Expiring / Temp Files](#expiring--temp-files) |
| `WithMaxFileSize(int64)` | Reject uploads larger than the limit (0 = no limit) |
| `WithMaxTrashBatchSize(int)` | Cap descendant files moved in a single folder delete |
| `WithBackendOpTimeout(time.Duration)` | Bound backend I/O time during a delete (negative = disable) |
| `MediaDir(string)` | Local filesystem media directory |
| `FlatNameSpaces(bool)` | Store files by UUID at the root vs. by logical path |

---

## Storage Backends

Buckt's `FileBackend` interface lets you swap storage providers without changing your application code. The cloud backends are separate Go modules, so you only pull in the SDKs you actually use.

| Backend | Module | Use Case |
|---|---|---|
| Local FS | (built in) | Development, single-server deployments |
| AWS S3 | `cloud/aws` | Production cloud, durable object storage |
| Cloudflare R2 | `cloud/aws` | S3-compatible, zero egress fees |
| Google Cloud Storage | `cloud/gcp` | GCP-native applications |
| Azure Blob | `cloud/azure` | Azure-native applications |
| MinIO / Ceph | `cloud/aws` | Self-hosted S3-compatible storage |

### Local Filesystem

The default. No configuration needed.

```go
  client, _ := buckt.Default()
```

### AWS S3

```go
  import (
    "github.com/Rhaqim/buckt"
    "github.com/Rhaqim/buckt/cloud/aws"
  )

  s3, err := aws.NewBackend(aws.Config{
    AccessKey: "AKIA...",
    SecretKey: "...",
    Region:    "us-east-1",
    Bucket:    "my-bucket",
  })
  if err := s3.Ping(ctx); err != nil { /* fail fast on bad credentials */ }

  client, _ := buckt.Default(buckt.WithBackend(s3))
```

### Cloudflare R2

R2 is S3-compatible. The `cloud/aws` backend auto-detects R2 from the endpoint suffix and configures path-style addressing.

```go
  r2, err := aws.NewBackend(aws.Config{
    AccessKey: "your-r2-access-key",
    SecretKey: "your-r2-secret",
    Bucket:    "my-bucket",
    Endpoint:  "https://<ACCOUNT_ID>.r2.cloudflarestorage.com",
    // Region auto-defaults to "auto" — leave empty
  })

  client, _ := buckt.Default(buckt.WithBackend(r2))
```

### Google Cloud Storage

```go
  import "github.com/Rhaqim/buckt/cloud/gcp"

  gcs, _ := gcp.NewBackend(gcp.Config{
    CredentialsFile: "service-account.json",
    Bucket:          "my-bucket",
  })

  client, _ := buckt.Default(buckt.WithBackend(gcs))
```

### Azure Blob Storage

```go
  import "github.com/Rhaqim/buckt/cloud/azure"

  az, _ := azure.NewBackend(azure.Config{
    AccountName: "myaccount",
    AccountKey:  "...",
    Container:   "my-container",
  })

  client, _ := buckt.Default(buckt.WithBackend(az))
```

All cloud backends expose a `Ping(ctx)` method for early connectivity validation — call it after `NewBackend` to surface credential or network issues at startup instead of on the first upload.

---

## Migration Between Backends

Need to move from local storage to S3 without downtime? Or from S3 to R2 to save on egress fees? Buckt's migration mode does dual-write reads from both backends with lazy forward migration on read.

```sh
              ┌───────────┐    Put   ┌──────────────┐
   Write ────►│  Primary  ├─────────►│  Secondary   │
              │ (current) │          │   (target)   │
              └─────┬─────┘          └──────┬───────┘
                    │                       │
                    │  Get: try primary,    │
                    │  fall back to         │
                    │  secondary if missing │
                    │                       │
                    └─────────► File ◄──────┘
```

```go
  import (
    "github.com/Rhaqim/buckt"
    "github.com/Rhaqim/buckt/cloud/aws"
  )

  s3, _ := aws.NewBackend(s3Config)

  client, _ := buckt.Default(buckt.WithMigration(buckt.MigrationConfig{
    From: buckt.LocalBackend(),  // current source of truth
    To:   s3,                    // target backend
  }))
```

**How it works:**

| Operation | Behavior |
|---|---|
| `Put` | Writes to both backends. Primary failure is a hard error; secondary failure is logged. |
| `Get` | Reads from primary first. If found, lazy-mirrors to secondary. If primary is missing, falls back to secondary. |
| `Delete` | Deletes from both backends. |
| `Move` | Moves in both backends. |

Migration is **always forward** (primary → secondary). The primary is treated as the source of truth and is never overwritten by secondary content.

**Bulk-copying pre-existing files.** Dual-write only mirrors *new* activity; files that predate the cutover are copied on demand as they're read. To migrate everything up front, call `MigrateAll` and poll `MigrationStatus`:

```go
  // Kick off a background copy of every stored object to the target.
  if err := client.MigrateAll(ctx); err != nil {
    log.Fatal(err) // ErrBackendUnavailable if the client wasn't built WithMigration
  }

  // Poll progress (completed == total when done). Idempotent: objects already
  // present in the target are skipped, so it's safe to re-run after a restart.
  for {
    done, total, _ := client.MigrationStatus(ctx)
    log.Printf("migrated %d/%d", done, total)
    if total > 0 && done >= total {
      break
    }
    time.Sleep(time.Second)
  }

  // completed counts every processed file, so the loop above always terminates
  // even if some files can't be copied. Check how many were left behind:
  if failed, _ := client.MigrationFailures(ctx); failed > 0 {
    log.Printf("%d file(s) failed after retries — fix the cause and re-run MigrateAll", failed)
  }
```

`MigrateAll` copies files **concurrently** with a bounded worker pool. Tune it for a high-latency target with `Concurrency` (default 8):

```go
  client, _ := buckt.Default(buckt.WithMigration(buckt.MigrationConfig{
    From:        buckt.LocalBackend(),
    To:          s3,
    Concurrency: 16, // more parallelism → faster, but more memory + API pressure
  }))
```

Each in-flight file is buffered in full, so higher concurrency trades memory and provider rate-limit headroom for throughput.

**Resumable.** In migration mode buckt records each copied object in the database (the `buckt_migration_models` table). If the process stops mid-migration, the next `MigrateAll` **resumes from where it left off** — already-copied files are skipped straight from the persisted state, without re-scanning the target. Progress (`MigrationStatus`) reflects the recorded state, so a restarted migration doesn't start its count from zero. Persistence is best-effort: a recording failure never fails a copy (copies are idempotent), and the state is keyed by the target backend's name.

`client.BackendName()` reports the active backend (`"local"`, `"s3"`, or `"local->s3"` mid-migration) — handy for a status badge. When you're done migrating, swap to the target alone:

```go
  // After migration completes
  client, _ := buckt.Default(buckt.WithBackend(s3)) // primary only
```

| Method | Description |
|---|---|
| `MigrateAll(ctx)` | Schedule a background copy of all pre-existing objects to the target (idempotent) |
| `MigrationStatus(ctx)` | Report `completed`/`total` processed and whether migration is enabled (`completed == total` when done) |
| `MigrationFailures(ctx)` | How many objects permanently failed after retries (re-run `MigrateAll` to retry them) |
| `BackendName()` | Name of the active backend (`local`, `s3`, or `local->s3`) |

---

## Image Derivatives

Generate resized image variants (thumbnails, medium sizes, …) from uploads. Define the variants once; the built-in pure-Go processor handles JPEG and PNG with no external dependencies.

```go
  client, _ := buckt.Default(
    buckt.WithImageDerivatives(
      buckt.DerivativeSpec{Name: "thumbnail", MaxWidth: 200},
      buckt.DerivativeSpec{Name: "medium", MaxWidth: 800},
    ),
  )

  // Generate the variants for a file (typically triggered from an upload event —
  // see Lifecycle Events to automate this).
  _ = client.GenerateDerivatives(fileID)

  // Fetch a variant back. Returns ErrNotFound if it hasn't been generated.
  data, contentType, err := client.GetDerivative(fileID, "thumbnail")
```

`DerivativeSpec` fields:

| Field | Description |
|---|---|
| `Name` | Variant name used to fetch it back (e.g. `"thumbnail"`) |
| `MaxWidth` | Max width in pixels; aspect ratio preserved, never upscaled |
| `Format` | Output encoding: `""` keeps the source format; `"jpeg"`/`"png"` built in; `"webp"` needs a matching processor |

**WebP** support lives in a separate module so the core stays dependency-free:

```bash
go get github.com/Rhaqim/buckt/imageproc/webp
```

```go
  import "github.com/Rhaqim/buckt/imageproc/webp"

  client, _ := buckt.Default(
    buckt.WithImageProcessor(webp.New()),
    buckt.WithImageDerivatives(
      buckt.DerivativeSpec{Name: "thumbnail", MaxWidth: 200, Format: "webp"},
    ),
  )
```

> Resizing runs inline with `GenerateDerivatives`. For heavy workloads, call it from an event handler that enqueues to a worker rather than blocking the upload.

---

## File Metadata

Attach arbitrary string key/value pairs to any file (stored as JSON on the file record):

```go
  _ = client.SetFileMetadata(fileID, map[string]string{
    "source": "web-ui",
    "album":  "vacation-2026",
  })

  meta, _ := client.GetFileMetadata(fileID) // map[string]string
```

---

## Deduplication

With `WithDedup()`, an upload whose bytes hash-match a file already present in the **same target folder** (for the same owner) returns that existing file's ID instead of writing the blob again — turning the "same photo sent four times" case into a single stored object.

```go
  client, _ := buckt.Default(buckt.WithDedup())
```

Scoped to the target folder, so it composes with nested-namespace mode and never resurrects a trashed duplicate. Off by default.

---

## Lifecycle Events

Register handlers to react to file operations. Handlers run **synchronously after** the operation commits — keep them fast (the intended pattern is to enqueue work and return). A panicking handler is recovered and never fails the originating call.

```go
  import "github.com/Rhaqim/buckt/pkg/events"

  onEvent := func(ctx context.Context, e events.Event) {
    if e.Type == events.FileUploaded {
      _ = client.GenerateDerivatives(e.FileID) // e.g. build thumbnails
    }
  }

  client, _ := buckt.Default(buckt.WithEventHandler(onEvent))
```

| Event Type | Fires when |
|---|---|
| `events.FileUploaded` | A new file's bytes are committed |
| `events.FileTrashed` | A file is moved to trash |
| `events.FileRestored` | A trashed file is restored |
| `events.FilePurged` | A file is hard-deleted |

The `events.Event` carries `Type`, `UserID`, `FileID`, `ParentID`, `Name`, `Path`, `ContentType`, `Size`, `Hash`, and `Time`.

> Events fire **after** the write, so they can't block an upload. To reject a file before it's stored, use [Upload Scanning](#upload-scanning).

---

## Upload Scanning

Register a scanner to inspect every upload's bytes **before** they're committed to the backend. Returning an error rejects the file — nothing is stored, no `file.uploaded` event fires, and the caller gets `ErrUploadRejected` wrapping your error.

buckt ships **no** scanning engine by design (keeping antivirus dependencies out of a storage library). You supply one — a ClamAV client, a VirusTotal lookup, a content-type allowlist, etc.

```go
  import "github.com/Rhaqim/buckt/pkg/scan"

  scanner := scan.ScannerFunc(func(ctx context.Context, name string, data []byte) error {
    if err := clamav.Scan(ctx, data); err != nil {
      return err // non-nil rejects the upload
    }
    return nil
  })

  client, _ := buckt.Default(buckt.WithUploadScanner(scanner))
```

Handle a rejection with `errors.Is` (map it to HTTP 422, for example):

```go
  _, err := client.UploadFile(userID, "", "invoice.pdf", "application/pdf", data)
  if errors.Is(err, buckt.ErrUploadRejected) {
    // rejected by the scanner — err also wraps the scanner's own reason
  }
```

The scanner runs at buckt's single upload chokepoint, so **every** upload path is covered — a caller can't forget to wire it in per call site.

---

## Metrics

Record per-backend operation metrics (count, errors, bytes, latency) with a pluggable recorder. The built-in in-memory collector has no dependencies:

```go
  import "github.com/Rhaqim/buckt/pkg/metrics"

  collector := metrics.NewCollector()
  client, _ := buckt.Default(buckt.WithMetrics(collector))

  // ... after some activity ...
  snap := collector.Snapshot() // map[backend]map[operation]metrics.Stat
  for backend, ops := range snap {
    for op, stat := range ops {
      fmt.Printf("%s.%s: %d ops, %d errors, %d bytes\n", backend, op, stat.Count, stat.Errors, stat.Bytes)
    }
  }
```

Each `metrics.Stat` holds `Count`, `Errors`, `Bytes`, and `TotalDur` (divide by `Count` for the mean latency). Implement the one-method `metrics.Recorder` interface to forward metrics to Prometheus, StatsD, or your own sink. Nil by default — zero overhead when unused.

---

## Expiring / Temp Files

Give a file a TTL and buckt will permanently delete it — blob, image derivatives, and metadata row — once it's due. Ideal for temp uploads, one-time shares, and scratch files.

```go
  fileID, _ := client.UploadFile("user123", "", "share.zip", "application/zip", data)

  client.SetFileTTL(fileID, 24*time.Hour)        // expire 24h from now
  // or an absolute time:
  client.SetFileExpiry(fileID, someTime)         // pass the zero time.Time to clear it
```

Expiry isn't automatic on its own — a **sweep** does the deleting, and you choose who drives it:

**App-driven (recommended for control / multi-instance):** call `PurgeExpired` from your own cron/worker.

```go
  purged, err := client.PurgeExpired(ctx) // permanently deletes everything past due; returns the count
```

**Built-in background sweeper (convenient for single-instance):** let buckt run the ticker for you.

```go
  client, _ := buckt.Default(buckt.WithExpirySweeper(10 * time.Minute))
  // buckt now calls PurgeExpired every 10 minutes; Client.Close() stops it.
```

Each expiry deletion emits a `file.purged` [event](#lifecycle-events), so you can hook post-deletion behaviour (audit log, notify, etc.). `PurgeExpired` works in batches and is safe to call repeatedly.

> **Beyond deletion:** because expiry rides on the events system, it's the first step toward general *scheduled actions* ("after 24h, email this file"). The delete action is built in; a future release can add app-registered actions on the same sweep. Ask if you need that now.

---

## Trash & Deletion

Buckt has two delete modes:

| Action | Result | API |
|---|---|---|
| **Move to Trash** | File/folder is moved to a per-user `__trash__` folder. Reversible. | `DeleteFile` / `DeleteFolder` |
| **Delete Permanently** | Hard-deleted from DB and storage backend. Irreversible. | `DeleteFilePermanently` / `DeleteFolderPermanently` |

The trash folder is a real folder in the user's account that's hidden from normal listings. Items keep their structure when trashed (folder hierarchy is preserved).

**Smart re-delete:** Calling `DeleteFile` on something that's already in trash will hard-delete it. This gives you a built-in "empty trash" mechanism without a separate API.

```go
  // Get trash contents
  trash, _ := client.GetTrashFolder("user123")
  for _, file := range trash.Files {
    fmt.Println("In trash:", file.Name)
  }

  // Restore: move it back out of trash
  client.MoveFile(fileID, originalParentID)

  // Or delete forever (since it's already in trash, this hard-deletes)
  client.DeleteFile(fileID)
```

When using a non-flat namespace, deleted items are physically moved on the storage backend so paths stay consistent — not just renamed in the database.

---

## Web Client

The optional web client gives you a Gin-based HTTP API and a Tailwind UI for free. It's a separate module so the core stays lean.

```go
  import (
    "github.com/Rhaqim/buckt"
    web "github.com/Rhaqim/buckt/client/web"
  )

  func main() {
    bucktClient, _ := buckt.Default()
    defer bucktClient.Close()

    router, _ := web.NewClient(bucktClient)
    router.Run(":8080")
  }
```

### Modes

| Mode | Routes Registered |
|---|---|
| `WebModeAll` (default) | UI at `/web` + API endpoints |
| `WebModeAPI` | API endpoints only |
| `WebModeUI` | UI at `/web` only |
| `WebModeMount` | API only, designed to be mounted onto a parent Gin engine |

### Features in the UI

- 📂 Breadcrumb navigation
- 🖼️ Image, video, audio, and PDF previews
- 🔄 Drag-and-drop to move files between folders
- 📝 Inline rename
- 🗂️ Folder browser for choosing move targets
- 🗑️ Move to Trash + Delete Permanently options
- ⌨️ Keyboard shortcuts (Esc to close modals)

[<img src="https://run.pstmn.io/button.svg" alt="Run In Postman" style="width: 128px; height: 32px;">](https://app.getpostman.com/run-collection/17061476-00806d0d-9584-4889-ade7-f8407932dba2?action=collection%2Ffork&source=rip_markdown&collection-url=entityId%3D17061476-00806d0d-9584-4889-ade7-f8407932dba2%26entityType%3Dcollection%26workspaceId%3D28697276-d953-482a-bd39-c4695366a55a)

---

## Database

By default Buckt uses SQLite (`./db.sqlite`). For production, bring your own connection.

| Driver | Status |
|---|---|
| SQLite | ✅ Built in, default |
| PostgreSQL | ✅ Tested |
| MySQL | 🟡 Planned |

```go
  import "database/sql"
  import _ "github.com/lib/pq"

  db, _ := sql.Open("postgres", "postgres://user:pass@host/db?sslmode=disable")

  client, _ := buckt.Default(buckt.WithDB(buckt.Postgres, db))
```

> Buckt does not close externally-provided connections — your application owns the lifecycle.

### Schema

Buckt manages two main tables via GORM AutoMigrate:

| Table | Purpose |
|---|---|
| `folder_models` | Folder hierarchy with parent_id, paths, and ownership |
| `file_models` | File metadata: name, size, content type, hash, owner |
| `buckt_migration_models` | (Migration mode only) Tracks per-file migration state |

---

## Caching

Two cache layers:

| Layer | Type | Purpose |
|---|---|---|
| **LRU File Cache** | In-memory (Ristretto) | Caches file bytes for hot reads |
| **Cache Manager** | Pluggable interface | Caches metadata (folders, file records) |

```go
  client, _ := buckt.Default(buckt.WithCache(buckt.CacheConfig{
    Manager: myRedisCache, // implements domain.CacheManager
    FileCacheConfig: buckt.FileCacheConfig{
      NumCounters: 10_000_000,    // 10M counters
      MaxCost:     1 << 30,       // 1GB
      BufferItems: 64,
    },
  }))
```

Implement the `CacheManager` interface to plug in Redis, Memcached, or any other cache:

```go
  type CacheManager interface {
    SetBucktValue(ctx context.Context, key string, value any) error
    GetBucktValue(ctx context.Context, key string) (any, error)
    DeleteBucktValue(ctx context.Context, key string) error
  }
```

---

## Security

Buckt ships with security defaults that catch common file-handling mistakes:

| Protection | Details |
|---|---|
| **Path traversal** | All file paths are validated against the media directory boundary |
| **File size limits** | Reject oversized uploads before allocating memory (`io.LimitReader`) |
| **Content sniffing** | Detects actual content type from bytes when client sends generic types |
| **Filename injection** | RFC 6266 percent-encoded `Content-Disposition` headers |
| **`X-Content-Type-Options: nosniff`** | Set on all file-serving endpoints |
| **Command injection guards** | Path validation before passing to `ffmpeg`/`convert` |
| **Self-move protection** | Folders can't be moved into themselves or their descendants |
| **Constraint integrity** | Unique constraints on `(user_id, parent_id, name)` prevent collisions |

---

## API Reference

### Folder Operations

```go
NewFolder(userID, parentID, name, description string) (string, error)
ListFolders(folderID string) ([]FolderModel, error)
GetFolderWithContent(userID, folderID string) (*FolderModel, error)
GetTrashFolder(userID string) (*FolderModel, error)
MoveFolder(userID, folderID, newParentID string) error
RenameFolder(userID, folderID, newName string) error
DeleteFolder(folderID string) (string, error)              // → trash
DeleteFolderPermanently(userID, folderID string) (string, error)  // → hard delete
```

### File Operations

```go
UploadFile(userID, parentID, name, contentType string, data []byte) (string, error)
UploadFileFromReader(userID, parentID, name, contentType string, r io.Reader) (string, error)
GetFile(fileID string) (*FileModel, error)
GetFileStream(fileID string) (*FileModel, io.ReadCloser, error)
ListFiles(folderID string) ([]FileModel, error)
ListFilesMetadata(folderID string) ([]FileModel, error)
MoveFile(fileID, newParentID string) error
DeleteFile(fileID string) (string, error)              // → trash
DeleteFilePermanently(fileID string) (string, error)   // → hard delete

// Metadata
SetFileMetadata(fileID string, metadata map[string]string) error
GetFileMetadata(fileID string) (map[string]string, error)

// Expiry / temp files
SetFileTTL(fileID string, ttl time.Duration) error       // expire ttl from now
SetFileExpiry(fileID string, at time.Time) error         // absolute; zero time clears
PurgeExpired(ctx context.Context) (purged int, err error) // delete everything past due

// Image derivatives
GenerateDerivatives(fileID string) error
GetDerivative(fileID, name string) (data []byte, contentType string, err error)

// Migration (only when built WithMigration)
MigrateAll(ctx context.Context) error
MigrationStatus(ctx context.Context) (completed, total int64, ok bool)
MigrationFailures(ctx context.Context) (failed int64, ok bool)
BackendName() string
```

Every method has a `*Context` variant that takes an explicit `context.Context` for cancellation and timeouts.

### Errors

Branch on failures with `errors.Is` using the re-exported sentinels (also available from `pkg/buckterr`):

| Sentinel | Meaning | Suggested HTTP status |
|---|---|---|
| `buckt.ErrNotFound` | File/folder (or derivative) doesn't exist | 404 |
| `buckt.ErrInvalidID` | ID is not a valid UUID | 400 |
| `buckt.ErrInvalidName` | Empty/unsafe file or folder name | 400 |
| `buckt.ErrAlreadyExists` | Name collision on create/move/rename | 409 |
| `buckt.ErrFileTooLarge` | Upload exceeds `WithMaxFileSize` | 413 |
| `buckt.ErrUploadRejected` | Rejected by an upload scanner | 422 |
| `buckt.ErrTrashBatchExceeded` | Folder delete exceeds the trash batch cap | 409 |
| `buckt.ErrBackendUnavailable` | Backend unreachable / feature not enabled | 503 |

---

## Examples

| Example | Description |
|---|---|
| [Direct usage](example/direct/main.go) | Programmatic API without HTTP |
| [API server](example/client/web/api/main.go) | HTTP API only |
| [UI server](example/client/web/ui/main.go) | Web UI dashboard |
| [Mounted on parent server](example/client/web/mount/main.go) | Embed in existing Gin app |
| [AWS S3 backend](example/cloud/aws/main.go) | Production S3 setup |
| [Cloudflare R2 backend](example/cloud/cloudflare/main.go) | R2 with auto-detection |
| [Google Cloud Storage](example/cloud/gcp/main.go) | GCS configuration |
| [Azure Blob Storage](example/cloud/azure/main.go) | Azure setup |
| [Full-featured UI](example/client/web/ui/main.go) | Metrics, dedup, derivatives, and upload events across `local`/`migrate`/`r2` modes |
| [Headless migration](example/migration/headless/main.go) | Bulk `MigrateAll` + progress polling without the UI |

---

## Development

The repo ships secret-scanning so credentials never land in git. Install the [pre-commit](https://pre-commit.com) hooks once:

```bash
pip install pre-commit   # or: brew install pre-commit
pre-commit install
```

Thereafter every commit runs [gitleaks](https://github.com/gitleaks/gitleaks) against the staged diff (config in `.gitleaks.toml`). The same scan runs in CI (`.github/workflows/secret-scan.yml`) as a server-side backstop. To scan on demand:

```bash
gitleaks git --redact             # scan committed history
gitleaks git --staged --redact    # scan what you're about to commit
```

Merges to `main` update a draft GitHub Release via [release-drafter](https://github.com/release-drafter/release-drafter); a maintainer reviews and publishes it to cut the tag.

---

## License

MIT — see [LICENCE](LICENCE) for details.

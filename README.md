# Buckt

> A flexible Go media storage library with pluggable cloud backends, built-in trash, dual-write migration, and an optional web UI.

[![Go Report Card](https://goreportcard.com/badge/github.com/Rhaqim/buckt)](https://goreportcard.com/report/github.com/Rhaqim/buckt)
[![GoDoc](https://godoc.org/github.com/Rhaqim/buckt?status.svg)](https://pkg.go.dev/github.com/Rhaqim/buckt)
[![License](https://img.shields.io/github/license/Rhaqim/buckt)](LICENCE)

Buckt is a media storage package for Go applications that need to upload, organize, and serve files without rewriting storage logic for every project. It works out of the box with the local filesystem and SQLite, and scales up to S3, GCS, Azure Blob, and Cloudflare R2 with a single line of config.

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
  - [License](#license)

---

## Features

| Feature | Description |
|---|---|
| 📁 **Folder Hierarchy** | Logical folder tree with rename, move, and recursive operations |
| ☁️ **Pluggable Backends** | Local FS, S3, GCS, Azure Blob, Cloudflare R2 — all behind one interface |
| 🔄 **Live Migration** | Dual-write between two backends with lazy forward migration on read |
| 🗑️ **Trash & Restore** | Soft-delete moves items to a per-user trash folder, hard-delete is one click away |
| 🌐 **Web UI** | Optional Gin-based dashboard with drag-and-drop, breadcrumbs, and previews |
| 🔌 **HTTP API** | RESTful endpoints for upload, download, stream, list, move, delete |
| 🗄️ **BYO Database** | SQLite by default, plug in any `*sql.DB` (Postgres tested) |
| ⚡ **In-Memory Cache** | LRU file cache + pluggable cache manager interface |
| 🛡️ **Security First** | Path traversal protection, content-type validation, file size limits, RFC 6266 headers |
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
| `WithCache(CacheConfig)` | Plug in a custom cache manager + tune the LRU file cache |
| `WithBackend(Backend)` | Set a single storage backend |
| `WithMigration(MigrationConfig)` | Dual-write migration between two backends |
| `WithMaxFileSize(int64)` | Reject uploads larger than the limit (0 = no limit) |
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

Migration is **always forward** (primary → secondary). The primary is treated as the source of truth and is never overwritten by secondary content. When you're done migrating, swap the backends:

```go
  // After migration completes
  client, _ := buckt.Default(buckt.WithBackend(s3)) // primary only
```

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
```

Every method has a `*Context` variant that takes an explicit `context.Context` for cancellation and timeouts.

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
| [Migration: local → S3](example/cloud/migration/main.go) | Dual-write migration mode |

---

## License

MIT — see [LICENCE](LICENCE) for details.

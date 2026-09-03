# Migrating to S3 + Presigned URLs — Integration Guide

A guide for an application that already uses **buckt** with **local filesystem**
storage and wants to:

1. **Move storage to AWS S3** with zero downtime, and
2. **Serve and (optionally) receive files directly to/from S3** using presigned
   URLs, so bytes don't have to pass through the application server.

It's written for a **Go backend** (where buckt lives) talking to an
**Astro/Svelte frontend**. The current shape is assumed to be:

```
browser  ──upload──▶  Go backend  ──▶  buckt  ──▶  storage (local today)
browser  ◀─download──  Go backend  ◀──  buckt  ◀──  storage
```

Nothing here changes buckt's **file ID** — it stays the source of truth your app
tracks files by, before and after every change below.

---

## Contents

- [Part 1 — Migrate local → S3](#part-1--migrate-local--s3)
- [Part 2 — Presigned downloads (serve direct from S3)](#part-2--presigned-downloads-serve-direct-from-s3)
- [Part 3 — Direct uploads (register / confirm)](#part-3--direct-uploads-register--confirm)
- [Error reference](#error-reference)
- [Recommended rollout order](#recommended-rollout-order)

---

## Part 1 — Migrate local → S3

This is **application-side config only** — buckt already does the work. Your
upload/download flow is unchanged during and after the migration, so every buckt
feature (hashing, dedup, image derivatives, upload scanning, metadata) keeps
working exactly as today.

### 1.1 Build the S3 backend

Real AWS S3 differs from Cloudflare R2 config: **`Region` is required** and there
is **no `Endpoint`** (that field is only for R2/MinIO-style S3-compatible stores).

```go
import (
    "context"

    "github.com/Rhaqim/buckt"
    "github.com/Rhaqim/buckt/cloud/aws"
)

func newS3() (buckt.Backend, error) {
    s3, err := aws.NewBackend(aws.Config{
        AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
        SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
        Region:    os.Getenv("AWS_REGION"),     // e.g. "us-east-1" — REQUIRED
        Bucket:    os.Getenv("AWS_S3_BUCKET"),
        // Endpoint: ""  ← leave empty for real AWS S3
    })
    if err != nil {
        return nil, err
    }
    // Fail fast on bad credentials / bucket, instead of on the first upload.
    if err := s3.Ping(context.Background()); err != nil {
        return nil, err
    }
    return s3, nil
}
```

### 1.2 Switch buckt into migration mode (dual-write)

Instead of `buckt.WithBackend(local)`, wrap both backends. Every **new** upload is
now written to **both** local and S3; reads fall back to S3 if local is missing a
file.

```go
s3, err := newS3()
// ...

client, err := buckt.Default(
    // ...your existing options (WithDB, WithImageDerivatives, WithEventHandler, ...)
    buckt.WithMigration(buckt.MigrationConfig{
        From:        buckt.LocalBackend(), // current source of truth
        To:          s3,                   // destination
        Concurrency: 16,                   // parallel copies for the bulk pass
    }),
)
```

Deploy this. You're now safely dual-writing with **no downtime**.

### 1.3 Bulk-copy the files that already exist on local disk

Dual-write only mirrors *new* activity. Copy the backlog once:

```go
if err := client.MigrateAll(ctx); err != nil {
    log.Fatal(err)
}
for {
    done, total, _ := client.MigrationStatus(ctx)
    log.Printf("migrated %d/%d", done, total)
    if total > 0 && done >= total {
        break
    }
    time.Sleep(time.Second)
}
if failed, _ := client.MigrationFailures(ctx); failed > 0 {
    log.Printf("%d file(s) failed — check logs and re-run MigrateAll before cutting over", failed)
}
```

`MigrateAll` is **concurrent, resumable across restarts, and retries transient
failures** — safe to re-run. A runnable template is in
[`example/migration/s3`](../example/migration/s3/main.go).

### 1.4 Cut over

Once `MigrationStatus` reports `done == total` and `MigrationFailures` is `0`,
redeploy with S3 as the only backend:

```go
client, err := buckt.Default(
    // ...same options...
    buckt.WithBackend(s3), // S3 only; local no longer written or read
)
```

Keep the local copies around for a while as a cheap backup if you like. Uploading
to S3 is **free ingress** (AWS doesn't charge to bring data in), so the migration
itself costs nothing in transfer — only per-request (Class A) charges.

> **Note on presigning during migration:** the presigned-URL features in Parts 2
> and 3 require a **single cloud backend**. While the client is in migration mode
> they return `ErrUnsupported`. So finish the cut-over (1.4) before relying on
> presigned URLs.

---

## Part 2 — Presigned downloads (serve direct from S3)

**This is the biggest cost win on S3.** S3 charges **egress** (data leaving to the
internet). Today, serving a file means S3 → your server → browser, so you pay
egress *and* your server does the work. A presigned GET URL makes it **S3 →
browser directly**: one hop, and your server is out of the byte path.

Minting the URL still goes through your backend (it needs your S3 credentials);
only the **download** bypasses it.

### 2.1 Backend

Two Client methods (each has a `...Context` variant):

```go
// Direct-download URL for the file's bytes, valid for the TTL:
url, err := client.PresignedURL(fileID, 15*time.Minute)

// Direct URL for a generated image derivative (thumbnail, medium, ...):
thumbURL, err := client.PresignedDerivativeURL(fileID, "thumbnail", 15*time.Minute)

if errors.Is(err, buckt.ErrUnsupported) {
    // Local backend or mid-migration — fall back to streaming (GetFileStream / /serve).
}
```

If you use the bundled web client, there's also an HTTP endpoint:

```
GET /presign/:file_id?ttl=15m   →   { "status": "success", "url": "https://<bucket>.s3..." }
```

(Returns `501` when the backend can't presign.) It sits behind the same API guard
as your other routes, so your existing auth applies.

### 2.2 Frontend (Astro/Svelte)

Swap image `src` / download links from the streaming route to a presigned URL.
Because URLs **expire**, mint them at render/request time — don't cache them
long-term.

```ts
// Fetch a fresh presigned URL for a file id.
async function presignedURL(fileId: string, ttl = "15m"): Promise<string> {
  const res = await fetch(`/presign/${fileId}?ttl=${ttl}`, {
    headers: { Authorization: `Bearer ${token}` }, // your app's auth
  });
  if (!res.ok) throw new Error(`presign failed: ${res.status}`);
  const { url } = await res.json();
  return url;
}
```

```svelte
<script lang="ts">
  export let fileId: string;
  let src = "";
  // For a thumbnail, point your backend handler at PresignedDerivativeURL instead.
  presignedURL(fileId).then((u) => (src = u));
</script>

{#if src}
  <img {src} alt="" loading="lazy" />
{/if}
```

**Tips**

- Keep TTLs short (minutes). A presigned URL grants access to anyone holding it,
  bypassing your auth, for its whole lifetime — so don't log them.
- For long-lived pages/galleries, re-mint on load rather than persisting URLs.
- For thumbnails, presign the **derivative** (`/serve/:id/derivative/:name` today
  streams; back a handler with `PresignedDerivativeURL` to serve those direct too).

---

## Part 3 — Direct uploads (register / confirm)

The reverse direction: let the browser upload **straight to S3**, so large files
never pass through your server. This is a **two-step handshake**, and the buckt
**file ID is returned in step 1**, so your app can start tracking the file
immediately.

> **Read this trade-off first.** On the direct path buckt **never sees the
> bytes**, so these uploads are **NOT deduplicated, NOT scanned, and NOT
> content-hashed**, and image derivatives are generated **lazily** at finalize.
> Also, on S3 uploads are **free ingress** — so this saves your **server's**
> bandwidth/CPU, not S3 fees. Use it for **large / opaque files** (video,
> archives, big PDFs). Keep the normal `UploadFile` flow (bytes through the
> backend) for anything that needs dedup/scan/derivatives inline (e.g. user
> images).

### 3.1 The flow

```
1. browser ──"I want to upload X"──▶ backend ──▶ buckt.PresignUpload
                                       returns { fileID, uploadURL }
2. browser ──PUT bytes──▶ S3          (directly, using uploadURL)
3. browser ──"done, fileID, size"──▶ backend ──▶ buckt.FinalizeUpload
```

Between steps 1 and 3 the file is **pending**: the row (and ID) exist, but the
file is hidden from listings and not readable until finalized.

### 3.2 Backend

```go
// Step 1 — reserve. Returns a stable file ID + a presigned PUT URL.
fileID, uploadURL, err := client.PresignUpload(userID, parentID, "video.mp4", "video/mp4", 15*time.Minute)
if errors.Is(err, buckt.ErrUnsupported) {
    // Local / migration — fall back to the normal upload endpoint.
}

// Step 3 — confirm (after the browser's PUT succeeds). Makes the file live and
// fires file.uploaded (so your derivative handler runs, reading bytes from S3).
_, err = client.FinalizeUpload(fileID, sizeInBytes)
```

Web endpoints (accept **form or JSON**), behind the API guard:

```
POST /upload/presign     body: { parent_id?, file_name, content_type?, ttl? }
                         →    { status, file_id, upload_url }

POST /upload/finalize    body: { file_id, size? }
                         →    { status, file_id }
```

### 3.3 Frontend (Astro/Svelte)

```ts
async function directUpload(file: File, parentId: string): Promise<string> {
  // 1. Reserve.
  const reserve = await fetch("/upload/presign", {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({
      parent_id: parentId,
      file_name: file.name,
      content_type: file.type,
      ttl: "15m",
    }),
  });
  if (!reserve.ok) throw new Error(`reserve failed: ${reserve.status}`);
  const { file_id, upload_url } = await reserve.json();

  // 2. Upload the bytes straight to S3. Set Content-Type so the stored object
  //    has the right type for later presigned downloads (otherwise S3 stores it
  //    as binary and browsers download instead of rendering).
  const put = await fetch(upload_url, {
    method: "PUT",
    headers: { "Content-Type": file.type },
    body: file,
  });
  if (!put.ok) throw new Error(`upload failed: ${put.status}`);

  // 3. Confirm.
  const finalize = await fetch("/upload/finalize", {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
    body: JSON.stringify({ file_id, size: file.size }),
  });
  if (!finalize.ok) throw new Error(`finalize failed: ${finalize.status}`);

  return file_id; // your app can now track / reference this file
}
```

**Important frontend detail:** send the `Content-Type` header on the PUT (step 2).
It isn't part of the presigned signature (so it's optional), but S3 stores
whatever you send — and that stored type is what a later presigned **download**
serves. Set it to the real MIME type so images/PDFs render inline instead of
downloading.

### 3.4 Cleaning up abandoned reservations

If a client reserves (step 1) but never finalizes (step 3), you're left with a
hidden pending row and possibly an orphan S3 object. Two easy guards:

- Set an **S3 lifecycle rule** to expire incomplete/old objects under your prefix.
- Periodically delete stale pending files from your side (they're hidden from
  listings but you hold the IDs you handed out).

---

## Error reference

Branch on these with `errors.Is`; they're re-exported from the root `buckt`
package. Suggested HTTP mappings are what the bundled web client uses.

| Error | Meaning | HTTP |
|---|---|---|
| `buckt.ErrUnsupported` | Backend can't do this (presign on local, or mid-migration) | 501 |
| `buckt.ErrNotFound` | File/derivative/object doesn't exist (e.g. finalize before the PUT landed) | 404 |
| `buckt.ErrUploadRejected` | An upload scanner rejected the file (normal upload path) | 422 |
| `buckt.ErrFileTooLarge` | Upload exceeds `WithMaxFileSize` | 413 |
| `buckt.ErrBackendUnavailable` | Backend unreachable / feature not enabled | 503 |

---

## Recommended rollout order

1. **Migrate to S3** (Part 1) with the current upload/download flow. Lowest risk;
   all features intact. Verify `MigrationFailures == 0`, then cut over.
2. **Adopt presigned downloads** (Part 2). This is the real S3 cost saver (egress)
   and offloads your server on the high-volume read side. Frontend change only —
   buckt already supports it.
3. **Adopt direct uploads selectively** (Part 3) — only for large/opaque files
   where server upload load actually hurts. Keep the normal `UploadFile` flow for
   images and anything needing dedup/scan/derivatives inline.

Do them in that order and each step is independently shippable and reversible.

// Package main demonstrates a zero-downtime migration from the local filesystem
// to an S3-compatible object store (AWS S3, Cloudflare R2, MinIO, …) headlessly:
//
//   - migration mode mirrors every write to BOTH backends and lazily copies
//     files forward as they're read;
//   - MigrateAll bulk-copies files that already existed before the cutover;
//   - MigrationStatus reports progress.
//
// For the UI-driven walkthrough of the same lifecycle, see ../ui.
//
// Configure the target from the environment (values shown for Cloudflare R2):
//
//	export R2_ACCESS_KEY=... R2_SECRET_KEY=... R2_BUCKET=... \
//	       R2_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com
//	go run ./migration/headless
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/cloud/aws"
)

func main() {
	target, err := newObjectStore()
	if err != nil {
		log.Fatalf("target backend: %v", err)
	}

	// Migration mode: local is the source of truth, the object store is the
	// destination. Reuses ./db.sqlite + ./media, so any files an earlier local
	// run stored are picked up.
	client, err := buckt.Default(
		buckt.WithLog(buckt.LogConfig{}),
		buckt.WithMigration(buckt.MigrationConfig{
			From: buckt.LocalBackend(),
			To:   target,
		}),
	)
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// A new upload in migration mode is dual-written to local AND the target.
	if _, err := client.UploadFile("demo", "", "hello.txt", "text/plain", []byte("hi from buckt")); err != nil {
		log.Fatalf("upload: %v", err)
	}
	fmt.Println("uploaded hello.txt (mirrored to both local and the object store)")

	// Bulk-copy every pre-existing file to the target, then watch progress.
	if err := client.MigrateAll(ctx); err != nil {
		log.Fatalf("migrate all: %v", err)
	}
	for {
		done, total, _ := client.MigrationStatus(ctx)
		fmt.Printf("migration: %d/%d objects copied\n", done, total)
		if total == 0 || done >= total {
			break
		}
		time.Sleep(time.Second)
	}
	fmt.Println("✅ migration complete — switch to buckt.WithBackend(target) to run on the object store only")
}

func newObjectStore() (buckt.Backend, error) {
	cfg := aws.Config{
		AccessKey: os.Getenv("R2_ACCESS_KEY"),
		SecretKey: os.Getenv("R2_SECRET_KEY"),
		Bucket:    os.Getenv("R2_BUCKET"),
		Endpoint:  os.Getenv("R2_ENDPOINT"),
		Region:    os.Getenv("R2_REGION"),
	}
	if os.Getenv("R2_USE_PATH_STYLE") == "true" {
		cfg.UsePathStyle = true // MinIO and other non-R2 S3-compatible stores
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("set R2_ACCESS_KEY, R2_SECRET_KEY, R2_BUCKET and R2_ENDPOINT")
	}

	backend, err := aws.NewBackend(cfg)
	if err != nil {
		return nil, err
	}
	if err := backend.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("cannot reach the object store: %w", err)
	}
	return backend, nil
}

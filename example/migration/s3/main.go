// Package main is a copy-paste template for migrating an app from local
// filesystem storage to AWS S3 with zero downtime — the "Phase 1" swap. It's
// application-side wiring: buckt already does all the work.
//
// The lifecycle:
//  1. Run in migration mode (local is the source of truth, S3 the target). Every
//     new upload dual-writes to both; reads fall back to S3.
//  2. MigrateAll bulk-copies the files that already exist on local disk to S3
//     (concurrent, resumable across restarts, retries transient failures).
//  3. Once MigrationStatus reports completion with zero MigrationFailures, cut
//     over: redeploy with buckt.WithBackend(s3) alone (drop local).
//
// Real AWS S3 config differs from Cloudflare R2: set Region, and DON'T set an
// Endpoint (that's the R2/MinIO override). Configure from the environment:
//
//	export AWS_REGION=us-east-1 AWS_S3_BUCKET=my-bucket \
//	       AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=...
//	go run ./migration/s3
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/cloud/aws"
)

func main() {
	s3, err := newS3Backend()
	if err != nil {
		log.Fatalf("s3 backend: %v", err)
	}

	// Migration mode: local (the app's current storage) is the source of truth,
	// S3 is the destination. Reuses ./db.sqlite + ./media, so files an earlier
	// local-only run stored are picked up by MigrateAll below.
	client, err := buckt.Default(
		buckt.WithLog(buckt.LogConfig{Silence: true}),
		buckt.WithMigration(buckt.MigrationConfig{
			From:        buckt.LocalBackend(),
			To:          s3,
			Concurrency: 16, // tune for throughput vs memory / S3 rate limits
		}),
	)
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	log.Printf("backend: %s (dual-writing new uploads; bulk-copying existing files)", client.BackendName())

	// Bulk-copy every pre-existing local file to S3. Safe to re-run after an
	// interruption — already-copied files are skipped from persisted state.
	if err := client.MigrateAll(ctx); err != nil {
		log.Fatalf("MigrateAll: %v", err)
	}
	for {
		done, total, _ := client.MigrationStatus(ctx)
		log.Printf("migrated %d/%d", done, total)
		if total > 0 && done >= total {
			break
		}
		if total == 0 {
			log.Println("no pre-existing files to copy")
			break
		}
		time.Sleep(time.Second)
	}

	if failed, _ := client.MigrationFailures(ctx); failed > 0 {
		log.Printf("⚠️ %d file(s) failed to copy — fix the cause and re-run before cutting over", failed)
		return
	}

	log.Println("✅ migration complete. Cut over by redeploying with:")
	log.Println("     buckt.Default(buckt.WithBackend(s3))   // S3 only, drop local")
}

// newS3Backend builds an AWS S3 backend from the environment. Note: Region is
// required and there is NO Endpoint (unlike R2/MinIO).
func newS3Backend() (buckt.Backend, error) {
	backend, err := aws.NewBackend(aws.Config{
		AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Region:    os.Getenv("AWS_REGION"),
		Bucket:    os.Getenv("AWS_S3_BUCKET"),
	})
	if err != nil {
		return nil, err
	}
	if err := backend.Ping(context.Background()); err != nil {
		return nil, err
	}
	return backend, nil
}

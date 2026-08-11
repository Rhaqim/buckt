// Package main runs the full-featured Buckt web UI (metrics, dedup, image
// derivatives, and a file.uploaded handler) over one of three storage modes,
// selected with -mode:
//
//	-mode=local    (default) filesystem storage in ./media + ./db.sqlite
//	-mode=migrate  dual-write local -> object store, bulk-copying existing files
//	-mode=r2       object store only
//
// All modes share the same features and the same ./db.sqlite + ./media, so you
// can start on local, upload files, then re-run with -mode=migrate to watch them
// migrate (the header badge shows the active backend and live progress).
//
// migrate/r2 read the object store from the environment (Cloudflare R2 shown):
//
//	export R2_ACCESS_KEY=... R2_SECRET_KEY=... R2_BUCKET=... \
//	       R2_ENDPOINT=https://<ACCOUNT_ID>.r2.cloudflarestorage.com
//	# for MinIO also: R2_USE_PATH_STYLE=true R2_REGION=us-east-1
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Rhaqim/buckt"
	"github.com/Rhaqim/buckt/client/web"
	"github.com/Rhaqim/buckt/cloud/aws"
	"github.com/Rhaqim/buckt/pkg/events"
	"github.com/Rhaqim/buckt/pkg/metrics"
)

func main() {
	mode := flag.String("mode", "local", "storage mode: local | migrate | r2")
	flagPort := flag.String("port", envOr("PORT", "8080"), "Port to run the server on")
	withDB := flag.Bool("db", false, "Use an external Postgres database")
	flatNamespaces := flag.Bool("flat", false, "Use flat namespaces")
	flag.Parse()

	// Feature set — identical across every storage mode.
	opts := []buckt.ConfigFunc{
		buckt.FlatNameSpaces(*flatNamespaces),
		buckt.WithMetrics(metrics.NewCollector()),
		buckt.WithDedup(),
		buckt.WithImageDerivatives(
			buckt.DerivativeSpec{Name: "thumbnail", MaxWidth: 200},
			buckt.DerivativeSpec{Name: "medium", MaxWidth: 800},
		),
	}

	if *withDB {
		db, err := initDB()
		if err != nil {
			log.Fatalf("Failed to initialize the database: %v", err)
		}
		opts = append(opts, buckt.WithDB(buckt.Postgres, db))
	}

	// Storage mode only swaps the backend — everything above stays the same.
	migrating := false
	switch *mode {
	case "local":
		log.Println("📁 storage: local filesystem (./media + ./db.sqlite)")
	case "migrate":
		store, err := newObjectStore()
		if err != nil {
			log.Fatalf("migrate mode: %v", err)
		}
		opts = append(opts, buckt.WithMigration(buckt.MigrationConfig{From: buckt.LocalBackend(), To: store}))
		migrating = true
		log.Println("🔄 storage: migration (local → object store; existing files bulk-copying)")
	case "r2":
		store, err := newObjectStore()
		if err != nil {
			log.Fatalf("r2 mode: %v", err)
		}
		opts = append(opts, buckt.WithBackend(store))
		log.Println("☁️  storage: object store only")
	default:
		log.Fatalf("unknown -mode %q (use local | migrate | r2)", *mode)
	}

	// client is referenced by the upload handler, so declare it first and assign
	// with '=' (not ':=') to keep the closure bound to this variable.
	var client *buckt.Client

	// onUpload fires after every successful upload: generate image derivatives
	// and tag the file. In production, enqueue this to a worker rather than
	// resizing inline.
	onUpload := func(_ context.Context, e events.Event) {
		if e.Type != events.FileUploaded {
			return
		}
		if err := client.GenerateDerivatives(e.FileID); err != nil {
			log.Printf("derivative generation failed for %s: %v", e.FileID, err)
		}
		if err := client.SetFileMetadata(e.FileID, map[string]string{"source": "web-ui"}); err != nil {
			log.Printf("setting metadata failed for %s: %v", e.FileID, err)
		}
	}
	opts = append(opts, buckt.WithEventHandler(onUpload))

	var err error
	client, err = buckt.Default(opts...)
	if err != nil {
		log.Fatalf("Failed to initialize Buckt: %v", err)
	}
	defer client.Close()

	// In migration mode, bulk-copy pre-existing files to the target in the
	// background while the UI keeps serving (new uploads are mirrored anyway).
	if migrating {
		go bulkMigrate(client)
	}

	webClient, err := web.NewClient(client)
	if err != nil {
		log.Fatalf("Failed to create web client: %v", err)
	}

	log.Printf("Serving the Buckt UI at http://localhost:%s/web/", *flagPort)
	if err := webClient.Run(":" + *flagPort); err != nil {
		log.Fatalf("Failed to start Buckt: %v", err)
	}
}

// bulkMigrate kicks off MigrateAll and logs progress until it finishes.
func bulkMigrate(client *buckt.Client) {
	ctx := context.Background()
	if err := client.MigrateAll(ctx); err != nil {
		log.Printf("could not start bulk migration: %v", err)
		return
	}
	for {
		done, total, _ := client.MigrationStatus(ctx)
		if total == 0 {
			log.Println("migration: no pre-existing files to copy")
			return
		}
		log.Printf("migration: %d/%d objects processed", done, total)
		if done >= total {
			if failed, _ := client.MigrationFailures(ctx); failed > 0 {
				log.Printf("⚠️ migration finished with %d failure(s) — check the logs and re-run to retry them", failed)
			} else {
				log.Println("✅ migration complete — you can now restart with -mode=r2")
			}
			return
		}
		time.Sleep(time.Second)
	}
}

// newObjectStore builds an S3-compatible backend (Cloudflare R2 by default) from
// environment variables.
func newObjectStore() (buckt.Backend, error) {
	cfg := aws.Config{
		AccessKey: os.Getenv("R2_ACCESS_KEY"),
		SecretKey: os.Getenv("R2_SECRET_KEY"),
		Bucket:    os.Getenv("R2_BUCKET"),
		Endpoint:  os.Getenv("R2_ENDPOINT"),
		Region:    os.Getenv("R2_REGION"),
	}
	if os.Getenv("R2_USE_PATH_STYLE") == "true" {
		cfg.UsePathStyle = true // for non-R2 S3-compatible stores like MinIO
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" || cfg.Bucket == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("set R2_ACCESS_KEY, R2_SECRET_KEY, R2_BUCKET and R2_ENDPOINT")
	}

	backend, err := aws.NewBackend(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create object-store backend: %w", err)
	}
	if err := backend.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("cannot reach the object store: %w", err)
	}
	return backend, nil
}

func initDB() (*sql.DB, error) {
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		"localhost", 5432, "postgres", "password", "postgres")
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the database: %w", err)
	}
	return db, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

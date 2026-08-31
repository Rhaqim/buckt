// Package main demonstrates buckt's expiring / temp-file API headlessly:
//
//   - SetFileTTL gives a file a time-to-live (it's permanently deleted when due);
//   - SetFileExpiry sets an absolute expiry (or clears it with the zero time);
//   - PurgeExpired deletes everything past due — call it from your own scheduler,
//     or use buckt.WithExpirySweeper to run it on a background ticker.
//
// For the UI-driven version (set a TTL from the file menu, watch the sweeper
// delete it), run the ui example with an interval:  make ui EXPIRY=30s
//
//	go run ./expiry/headless
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Rhaqim/buckt"
)

func main() {
	// Local filesystem storage (./media + ./db.sqlite) is all we need to show
	// the lifecycle. Silence the banner logs to keep the output readable.
	client, err := buckt.Default(buckt.WithLog(buckt.LogConfig{Silence: true}))
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	const user = "demo"

	// 1. Upload a "permanent" file and a "temp" file.
	keep, err := client.UploadFileContext(ctx, user, "", "keep.txt", "text/plain", []byte("I stay"))
	if err != nil {
		log.Fatalf("upload keep: %v", err)
	}
	// Upload the temp file WITH its TTL in one call — the expiry is written in
	// the same insert (a "save temp"), so there's no window where it exists
	// without a TTL. In real code the TTL might be minutes or days; we use 2s so
	// the demo finishes quickly. (SetFileTTL adds/changes a TTL after upload;
	// UploadFileWithExpiry takes an absolute time instead.)
	temp, err := client.UploadFileWithTTLContext(ctx, user, "", "temp.txt", "text/plain", []byte("delete me soon"), 2*time.Second)
	if err != nil {
		log.Fatalf("upload temp: %v", err)
	}
	log.Printf("uploaded keep=%s (no expiry) and temp=%s (TTL 2s, set at upload)", keep, temp)

	// 3. A purge right now removes nothing — the TTL hasn't elapsed.
	n, err := client.PurgeExpired(ctx)
	if err != nil {
		log.Fatalf("purge: %v", err)
	}
	log.Printf("purge before expiry: %d file(s) removed (expected 0)", n)

	// 4. Wait for the TTL to pass, then purge again.
	time.Sleep(3 * time.Second)
	n, err = client.PurgeExpired(ctx)
	if err != nil {
		log.Fatalf("purge: %v", err)
	}
	log.Printf("purge after expiry: %d file(s) removed (expected 1)", n)

	// 5. The temp file is gone; the permanent file remains.
	if _, err := client.GetFileContext(ctx, temp); errors.Is(err, buckt.ErrNotFound) {
		log.Printf("✅ temp.txt was purged")
	} else {
		log.Printf("⚠️ temp.txt unexpectedly still present (err=%v)", err)
	}
	if _, err := client.GetFileContext(ctx, keep); err == nil {
		log.Printf("✅ keep.txt is still here")
	} else {
		log.Printf("⚠️ keep.txt unexpectedly missing: %v", err)
	}

	log.Println("Tip: instead of calling PurgeExpired yourself, pass " +
		"buckt.WithExpirySweeper(interval) and buckt runs it in the background.")
}

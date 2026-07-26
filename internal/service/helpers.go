package service

import (
	"context"
	"time"
)

// withBackendTimeout bounds the time spent on backend I/O so a DB transaction
// can't stay open indefinitely waiting on slow storage. A non-positive timeout
// means no bound (a no-op cancel is returned). Shared by the file and folder
// services, which both apply it around backend calls made inside a transaction.
func withBackendTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, timeout)
}

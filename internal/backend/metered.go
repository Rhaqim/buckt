package backend

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Rhaqim/buckt/internal/domain"
	"github.com/Rhaqim/buckt/pkg/buckterr"
	"github.com/Rhaqim/buckt/pkg/metrics"
)

// meteredBackend wraps a FileBackend and records one metrics.Op per operation
// via the configured Recorder. It counts LOGICAL operations (one per method
// call); a backend that also implements opRecorderSetter may additionally record
// its own per-API-call metrics.
type meteredBackend struct {
	inner domain.FileBackend
	rec   metrics.Recorder
}

// opRecorderSetter is the structural contract buckt checks for so a backend can
// emit per-API-call metrics WITHOUT importing buckt. The callback signature uses
// only standard-library types on purpose: a backend satisfies this by declaring
// a matching method (e.g. the S3 backend's SetOpRecorder), keeping the
// dependency one-way (buckt → backends, never the reverse).
type opRecorderSetter interface {
	SetOpRecorder(record func(op string, bytes int64, dur time.Duration, err error))
}

// NewMeteredBackend wraps inner so each call is recorded to rec. If rec is nil,
// inner is returned unwrapped (zero overhead). If inner implements
// opRecorderSetter, buckt injects a callback so the backend can emit its own
// finer-grained (per-API-call) metrics into the same recorder.
func NewMeteredBackend(inner domain.FileBackend, rec metrics.Recorder) domain.FileBackend {
	if rec == nil || inner == nil {
		return inner
	}
	if setter, ok := inner.(opRecorderSetter); ok {
		name := inner.Name()
		setter.SetOpRecorder(func(op string, bytes int64, dur time.Duration, err error) {
			rec.RecordOp(metrics.Op{Backend: name, Operation: op, Bytes: bytes, Duration: dur, Err: err})
		})
	}
	return &meteredBackend{inner: inner, rec: rec}
}

func (m *meteredBackend) record(op string, bytes int64, start time.Time, err error) {
	m.rec.RecordOp(metrics.Op{
		Backend:   m.inner.Name(),
		Operation: op,
		Bytes:     bytes,
		Duration:  time.Since(start),
		Err:       err,
	})
}

func (m *meteredBackend) Name() string { return m.inner.Name() }

func (m *meteredBackend) Put(ctx context.Context, path string, data []byte) error {
	start := time.Now()
	err := m.inner.Put(ctx, path, data)
	m.record(metrics.OpPut, int64(len(data)), start, err)
	return err
}

func (m *meteredBackend) Get(ctx context.Context, path string) ([]byte, error) {
	start := time.Now()
	data, err := m.inner.Get(ctx, path)
	m.record(metrics.OpGet, int64(len(data)), start, err)
	return data, err
}

func (m *meteredBackend) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	// Records the read operation (one provider GET) at call time. Streamed byte
	// counts aren't tracked here (they're only known after the caller finishes
	// reading); the operation count is what maps to R2 Class B billing.
	start := time.Now()
	rc, err := m.inner.Stream(ctx, path)
	m.record(metrics.OpStream, 0, start, err)
	return rc, err
}

func (m *meteredBackend) List(ctx context.Context, prefix string) ([]string, error) {
	start := time.Now()
	out, err := m.inner.List(ctx, prefix)
	m.record(metrics.OpList, 0, start, err)
	return out, err
}

func (m *meteredBackend) Exists(ctx context.Context, path string) (bool, error) {
	start := time.Now()
	ok, err := m.inner.Exists(ctx, path)
	m.record(metrics.OpExists, 0, start, err)
	return ok, err
}

func (m *meteredBackend) Move(ctx context.Context, oldPath, newPath string) error {
	start := time.Now()
	err := m.inner.Move(ctx, oldPath, newPath)
	m.record(metrics.OpMove, 0, start, err)
	return err
}

func (m *meteredBackend) Delete(ctx context.Context, path string) error {
	start := time.Now()
	err := m.inner.Delete(ctx, path)
	m.record(metrics.OpDelete, 0, start, err)
	return err
}

func (m *meteredBackend) DeleteFolder(ctx context.Context, prefix string) error {
	start := time.Now()
	err := m.inner.DeleteFolder(ctx, prefix)
	m.record(metrics.OpDeleteFolder, 0, start, err)
	return err
}

// PresignGetURL forwards to the inner backend when it supports presigning, so
// the capability survives the metering wrapper. Returns ErrUnsupported when the
// inner backend can't presign (e.g. local). Not metered: GET presigning is a
// local signing operation (any existence check the inner backend does is counted
// by its own recorder).
func (m *meteredBackend) PresignGetURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	p, ok := m.inner.(domain.PresignBackend)
	if !ok {
		return "", fmt.Errorf("backend %q does not support presigned URLs: %w", m.inner.Name(), buckterr.ErrUnsupported)
	}
	return p.PresignGetURL(ctx, key, ttl)
}

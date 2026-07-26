package backend

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/Rhaqim/buckt/pkg/metrics"
)

// fakeBackend is a minimal FileBackend that also implements the structural
// SetOpRecorder hook (matching what buckt injects into), so we can verify both
// the logical decorator and per-API-call injection without a real cloud client.
type fakeBackend struct {
	rec func(op string, bytes int64, dur time.Duration, err error)
}

func (f *fakeBackend) Name() string { return "fake" }
func (f *fakeBackend) SetOpRecorder(rec func(op string, bytes int64, dur time.Duration, err error)) {
	f.rec = rec
}

// Put also emits a per-API-call record to prove injection works end to end.
func (f *fakeBackend) Put(ctx context.Context, path string, data []byte) error {
	if f.rec != nil {
		f.rec("fake:PutObject", int64(len(data)), time.Millisecond, nil)
	}
	return nil
}
func (f *fakeBackend) Get(ctx context.Context, path string) ([]byte, error) { return []byte("x"), nil }
func (f *fakeBackend) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, nil
}
func (f *fakeBackend) List(ctx context.Context, prefix string) ([]string, error) { return nil, nil }
func (f *fakeBackend) Exists(ctx context.Context, path string) (bool, error)     { return true, nil }
func (f *fakeBackend) Delete(ctx context.Context, path string) error             { return nil }
func (f *fakeBackend) DeleteFolder(ctx context.Context, prefix string) error     { return nil }
func (f *fakeBackend) Move(ctx context.Context, oldPath, newPath string) error   { return nil }

func TestMeteredBackend_LogicalAndPerAPI(t *testing.T) {
	col := metrics.NewCollector()
	b := NewMeteredBackend(&fakeBackend{}, col)

	if err := b.Put(context.Background(), "p", []byte("hello")); err != nil {
		t.Fatal(err)
	}

	snap := col.Snapshot()["fake"]
	if snap == nil {
		t.Fatalf("no metrics recorded for fake backend: %v", col.Snapshot())
	}
	// Logical op from the decorator.
	if snap[metrics.OpPut].Count != 1 || snap[metrics.OpPut].Bytes != 5 {
		t.Errorf("logical put stat wrong: %+v", snap[metrics.OpPut])
	}
	// Per-API-call op injected via SetOpRecorder.
	if snap["fake:PutObject"].Count != 1 || snap["fake:PutObject"].Bytes != 5 {
		t.Errorf("per-API put stat wrong: %+v", snap["fake:PutObject"])
	}
}

func TestNewMeteredBackend_NilRecorderReturnsInner(t *testing.T) {
	inner := &fakeBackend{}
	if got := NewMeteredBackend(inner, nil); got != inner {
		t.Error("nil recorder should return the inner backend unwrapped")
	}
}

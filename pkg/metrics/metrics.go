// Package metrics provides buckt's built-in, dependency-free operation metrics.
//
// A Recorder receives one Op per storage-backend operation. buckt wraps every
// configured backend (local, S3, Azure, GCP, and the migration composite) in a
// metering decorator that emits logical operations (put/get/list/...). A backend
// may additionally emit finer-grained, per-API-call ops (e.g. the S3 backend
// emits one Op per real S3 request, capturing pagination) by declaring a
// SetOpRecorder(func(op string, bytes int64, dur time.Duration, err error))
// method — buckt detects it structurally and injects a callback, so the backend
// never imports buckt and the dependency stays one-way.
//
// The built-in Collector is an in-memory Recorder with no third-party
// dependencies — construct one, pass it via buckt.WithMetrics, and read counts
// with Snapshot:
//
//	m := metrics.NewCollector()
//	client, _ := buckt.New(cfg, buckt.WithMetrics(m))
//	// ... use client ...
//	for backend, ops := range m.Snapshot() {
//		for op, stat := range ops {
//			fmt.Printf("%s %s: %d ops, %d bytes\n", backend, op, stat.Count, stat.Bytes)
//		}
//	}
package metrics

import (
	"sync"
	"time"
)

// Logical operation names emitted by the metering decorator (one per FileBackend
// method). Backends emitting per-API-call metrics use "s3:<API>"-style names.
const (
	OpPut          = "put"
	OpGet          = "get"
	OpStream       = "stream"
	OpList         = "list"
	OpExists       = "exists"
	OpMove         = "move"
	OpDelete       = "delete"
	OpDeleteFolder = "delete_folder"
)

// Op is a single recorded backend operation.
type Op struct {
	Backend   string        // backend Name(), e.g. "s3", "local"
	Operation string        // logical ("put") or API-level ("s3:PutObject")
	Bytes     int64         // bytes written (Put) or read (Get); 0 otherwise
	Duration  time.Duration // wall-clock time of the operation
	Err       error         // non-nil if the operation failed
}

// Recorder receives operation records. Implementations MUST be safe for
// concurrent use and SHOULD keep RecordOp cheap and non-blocking, since it runs
// on the hot path of every backend call.
type Recorder interface {
	RecordOp(Op)
}

// Stat aggregates all records for one (backend, operation) pair.
type Stat struct {
	Count    int64         // number of operations
	Errors   int64         // number that returned a non-nil error
	Bytes    int64         // total bytes transferred
	TotalDur time.Duration // summed duration (divide by Count for the mean)
}

// Collector is a built-in in-memory Recorder. It is safe for concurrent use and
// has no dependencies beyond the standard library.
type Collector struct {
	mu   sync.Mutex
	byOp map[string]map[string]*Stat // backend -> operation -> stat
}

// NewCollector returns an empty Collector ready to pass to buckt.WithMetrics.
func NewCollector() *Collector {
	return &Collector{byOp: make(map[string]map[string]*Stat)}
}

// RecordOp implements Recorder.
func (c *Collector) RecordOp(op Op) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ops := c.byOp[op.Backend]
	if ops == nil {
		ops = make(map[string]*Stat)
		c.byOp[op.Backend] = ops
	}
	s := ops[op.Operation]
	if s == nil {
		s = &Stat{}
		ops[op.Operation] = s
	}
	s.Count++
	if op.Err != nil {
		s.Errors++
	}
	s.Bytes += op.Bytes
	s.TotalDur += op.Duration
}

// Snapshot returns a deep copy of the current counters: backend -> operation ->
// Stat. The returned maps are independent of the Collector and safe to read
// without locking.
func (c *Collector) Snapshot() map[string]map[string]Stat {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make(map[string]map[string]Stat, len(c.byOp))
	for be, ops := range c.byOp {
		m := make(map[string]Stat, len(ops))
		for op, s := range ops {
			m[op] = *s
		}
		out[be] = m
	}
	return out
}

// Reset clears all counters. Useful for exporting per-billing-period totals.
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.byOp = make(map[string]map[string]*Stat)
}

// Cloudflare R2 billing classes.
const (
	R2ClassA    = "A"    // writes/lists — the more expensive class
	R2ClassB    = "B"    // reads
	R2ClassFree = "free" // deletes (not billed by R2)
)

// R2Class classifies an operation name into its Cloudflare R2 billing class
// ("A", "B", or "free"), or "" if unknown. It recognizes both the logical
// operation names from the metering decorator and the "s3:<API>" names from the
// S3 backend's per-API-call metrics.
//
// The classification follows R2's documented operation classes. For the logical
// names it reflects the dominant billed call — e.g. "move" is Class A because it
// issues a CopyObject (Class A) plus a free DeleteObject. For exact billing use
// the "s3:<API>" names, which map one-to-one to R2 operations. R2's classes can
// change; treat this as a best-effort estimate, not a contract.
func R2Class(op string) string {
	switch op {
	case OpPut, OpList, OpMove, OpDeleteFolder,
		"s3:PutObject", "s3:CopyObject", "s3:ListObjectsV2",
		"s3:CreateMultipartUpload", "s3:CompleteMultipartUpload", "s3:UploadPart":
		return R2ClassA
	case OpGet, OpStream, OpExists,
		"s3:GetObject", "s3:HeadObject", "s3:HeadBucket":
		return R2ClassB
	case OpDelete, "s3:DeleteObject", "s3:DeleteObjects":
		return R2ClassFree
	default:
		return ""
	}
}

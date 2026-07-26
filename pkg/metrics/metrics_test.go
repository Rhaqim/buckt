package metrics

import (
	"errors"
	"testing"
	"time"
)

func TestCollector(t *testing.T) {
	c := NewCollector()

	c.RecordOp(Op{Backend: "s3", Operation: OpPut, Bytes: 100, Duration: time.Millisecond})
	c.RecordOp(Op{Backend: "s3", Operation: OpPut, Bytes: 50, Duration: time.Millisecond})
	c.RecordOp(Op{Backend: "s3", Operation: OpGet, Bytes: 200, Duration: 2 * time.Millisecond, Err: errors.New("boom")})
	c.RecordOp(Op{Backend: "local", Operation: OpDelete})

	snap := c.Snapshot()

	put := snap["s3"][OpPut]
	if put.Count != 2 || put.Bytes != 150 || put.Errors != 0 || put.TotalDur != 2*time.Millisecond {
		t.Errorf("s3 put stat wrong: %+v", put)
	}
	get := snap["s3"][OpGet]
	if get.Count != 1 || get.Bytes != 200 || get.Errors != 1 {
		t.Errorf("s3 get stat wrong: %+v", get)
	}
	if snap["local"][OpDelete].Count != 1 {
		t.Errorf("local delete count wrong: %+v", snap["local"][OpDelete])
	}

	// Snapshot must be an independent copy.
	snap["s3"][OpPut] = Stat{Count: 999}
	if c.Snapshot()["s3"][OpPut].Count != 2 {
		t.Error("Snapshot is not an independent copy")
	}

	c.Reset()
	if len(c.Snapshot()) != 0 {
		t.Error("Reset did not clear counters")
	}
}

func TestR2Class(t *testing.T) {
	classA := []string{OpPut, OpList, OpMove, OpDeleteFolder, "s3:PutObject", "s3:CopyObject", "s3:ListObjectsV2"}
	for _, op := range classA {
		if got := R2Class(op); got != R2ClassA {
			t.Errorf("R2Class(%q) = %q, want A", op, got)
		}
	}
	classB := []string{OpGet, OpStream, OpExists, "s3:GetObject", "s3:HeadObject"}
	for _, op := range classB {
		if got := R2Class(op); got != R2ClassB {
			t.Errorf("R2Class(%q) = %q, want B", op, got)
		}
	}
	free := []string{OpDelete, "s3:DeleteObject", "s3:DeleteObjects"}
	for _, op := range free {
		if got := R2Class(op); got != R2ClassFree {
			t.Errorf("R2Class(%q) = %q, want free", op, got)
		}
	}
	if got := R2Class("something-unknown"); got != "" {
		t.Errorf("R2Class(unknown) = %q, want empty", got)
	}
}

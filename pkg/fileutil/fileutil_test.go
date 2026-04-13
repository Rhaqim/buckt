package fileutil

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"
)

// makeMultipartFile creates a *multipart.FileHeader containing the given data,
// suitable for testing ProcessFile / ProcessFileWithLimit.
func makeMultipartFile(t *testing.T, name string, data []byte) *multipart.FileHeader {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+name+`"`)
	part, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("CreatePart: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("part.Write: %v", err)
	}
	w.Close()

	mr := multipart.NewReader(&buf, w.Boundary())
	form, err := mr.ReadForm(int64(len(data) + 1024))
	if err != nil {
		t.Fatalf("ReadForm: %v", err)
	}
	files := form.File["file"]
	if len(files) == 0 {
		t.Fatalf("no files parsed")
	}
	return files[0]
}

func TestProcessFile_ReadsAllBytes(t *testing.T) {
	data := []byte("hello world")
	fh := makeMultipartFile(t, "hello.txt", data)

	name, got, err := ProcessFile(fh)
	if err != nil {
		t.Fatalf("ProcessFile: %v", err)
	}
	if name != "hello.txt" {
		t.Errorf("filename = %q, want %q", name, "hello.txt")
	}
	if !bytes.Equal(got, data) {
		t.Errorf("data = %q, want %q", got, data)
	}
}

func TestProcessFileWithLimit_NoLimit(t *testing.T) {
	data := bytes.Repeat([]byte("a"), 1024)
	fh := makeMultipartFile(t, "blob", data)

	_, got, err := ProcessFileWithLimit(fh, 0) // 0 = no limit
	if err != nil {
		t.Fatalf("expected success with no limit, got: %v", err)
	}
	if len(got) != 1024 {
		t.Errorf("len = %d, want 1024", len(got))
	}
}

func TestProcessFileWithLimit_AtMax(t *testing.T) {
	const max = 100
	data := bytes.Repeat([]byte("x"), max)
	fh := makeMultipartFile(t, "ok", data)

	_, got, err := ProcessFileWithLimit(fh, max)
	if err != nil {
		t.Fatalf("expected success at exactly max, got: %v", err)
	}
	if len(got) != max {
		t.Errorf("len = %d, want %d", len(got), max)
	}
}

func TestProcessFileWithLimit_OverMax(t *testing.T) {
	const max = 100
	data := bytes.Repeat([]byte("x"), max+1)
	fh := makeMultipartFile(t, "too-big", data)

	_, _, err := ProcessFileWithLimit(fh, max)
	if err == nil {
		t.Fatal("expected error for max+1 bytes, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Errorf("error should mention 'exceeds', got: %v", err)
	}
}

func TestProcessFileWithLimit_RejectsByHeaderSize(t *testing.T) {
	// Even if the actual data is small, the multipart header.Size should
	// trigger the early rejection — but in our test helper data length
	// equals header size, so this verifies the early check fires.
	const max = 10
	data := bytes.Repeat([]byte("y"), 50)
	fh := makeMultipartFile(t, "big", data)

	_, _, err := ProcessFileWithLimit(fh, max)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSafeContentDisposition_PlainAscii(t *testing.T) {
	got := SafeContentDisposition("attachment", "hello.txt")
	want := "attachment; filename*=UTF-8''hello.txt"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSafeContentDisposition_StripsSlashes(t *testing.T) {
	got := SafeContentDisposition("attachment", "../../etc/passwd")
	if strings.Contains(got, "/") {
		t.Errorf("forward slash should be stripped, got: %q", got)
	}
}

func TestSafeContentDisposition_StripsBackslashes(t *testing.T) {
	got := SafeContentDisposition("attachment", `..\..\windows\system32`)
	if strings.Contains(got, `\`) {
		t.Errorf("backslash should be stripped, got: %q", got)
	}
}

func TestSafeContentDisposition_EscapesUnicode(t *testing.T) {
	// Unicode characters should be percent-encoded
	got := SafeContentDisposition("attachment", "café.txt")
	if strings.Contains(got, "café") {
		t.Errorf("unicode should be percent-encoded, got: %q", got)
	}
	if !strings.Contains(got, "%") {
		t.Errorf("expected percent-encoding, got: %q", got)
	}
}

func TestSafeContentDisposition_EscapesCRLF(t *testing.T) {
	// CR and LF must be escaped to prevent header injection
	got := SafeContentDisposition("attachment", "evil\r\nX-Injected: yes")
	if strings.ContainsAny(got, "\r\n") {
		t.Errorf("CRLF must not appear in output, got: %q", got)
	}
}

func TestSafeContentDisposition_InlineDisposition(t *testing.T) {
	got := SafeContentDisposition("inline", "image.png")
	if !strings.HasPrefix(got, "inline; ") {
		t.Errorf("expected 'inline' disposition prefix, got: %q", got)
	}
}

func TestSafeContentDisposition_RFC6266Format(t *testing.T) {
	// Verify the filename* parameter format matches RFC 5987 / 6266
	got := SafeContentDisposition("attachment", "test.pdf")
	if !strings.Contains(got, "filename*=UTF-8''") {
		t.Errorf("expected RFC 5987 filename* format, got: %q", got)
	}
}

// Sanity check that ProcessFile delegates correctly to ProcessFileWithLimit(0).
func TestProcessFile_DelegatesNoLimit(t *testing.T) {
	const want = 4096
	data := bytes.Repeat([]byte("z"), want)
	fh := makeMultipartFile(t, "x", data)

	_, got, err := ProcessFile(fh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != want {
		t.Errorf("len = %d, want %d", len(got), want)
	}
}

// Verify that io.LimitReader is doing its job for non-trivial limits.
func TestProcessFileWithLimit_BoundedRead(t *testing.T) {
	const max = 1024
	// 2x the limit
	data := bytes.Repeat([]byte("a"), 2*max)
	fh := makeMultipartFile(t, "x", data)

	_, _, err := ProcessFileWithLimit(fh, max)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
}

var _ io.Reader = (*bytes.Reader)(nil) // import sanity

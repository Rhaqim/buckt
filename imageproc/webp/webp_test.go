package webp

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	_ "golang.org/x/image/webp" // decoder, so DecodeConfig recognises our output
)

func srcPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 50, A: 255})
		}
	}
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestResize_EncodesWebP(t *testing.T) {
	out, ct, err := New().Resize(srcPNG(t, 300, 200), 100, "webp")
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/webp" {
		t.Fatalf("content type = %q, want image/webp", ct)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("output is not a decodable image: %v", err)
	}
	if format != "webp" {
		t.Fatalf("format = %q, want webp", format)
	}
	if cfg.Width != 100 {
		t.Fatalf("width = %d, want 100", cfg.Width)
	}
}

func TestResize_KeepsSourceAndTranscodes(t *testing.T) {
	// "" keeps the source (png), never upscales.
	out, ct, err := New().Resize(srcPNG(t, 120, 120), 999, "")
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/png" {
		t.Fatalf("content type = %q, want image/png", ct)
	}
	if cfg, _, _ := image.DecodeConfig(bytes.NewReader(out)); cfg.Width != 120 {
		t.Fatalf("width = %d, want 120 (no upscale)", cfg.Width)
	}

	// explicit jpeg transcode
	if _, ct, err := New().Resize(srcPNG(t, 200, 200), 50, "jpeg"); err != nil || ct != "image/jpeg" {
		t.Fatalf("jpeg transcode: ct=%q err=%v", ct, err)
	}
}

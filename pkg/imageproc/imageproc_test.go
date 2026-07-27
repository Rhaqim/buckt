package imageproc

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func makeImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	return img
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, makeImage(w, h)); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func widthOf(t *testing.T, data []byte) int {
	t.Helper()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Width
}

func TestDefault_ResizeKeepsSourceFormatAndScales(t *testing.T) {
	out, ct, err := Default().Resize(pngBytes(t, 400, 200), 100, "")
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/png" {
		t.Fatalf("content type = %q, want image/png", ct)
	}
	if w := widthOf(t, out); w != 100 {
		t.Fatalf("width = %d, want 100", w)
	}
}

func TestDefault_NeverUpscales(t *testing.T) {
	out, _, err := Default().Resize(pngBytes(t, 120, 120), 999, "")
	if err != nil {
		t.Fatal(err)
	}
	if w := widthOf(t, out); w != 120 {
		t.Fatalf("width = %d, want 120 (no upscaling)", w)
	}
}

func TestDefault_TranscodeToJPEG(t *testing.T) {
	out, ct, err := Default().Resize(pngBytes(t, 200, 200), 50, "jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if ct != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", ct)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("output is not valid JPEG: %v", err)
	}
}

func TestDefault_UnsupportedFormatErrors(t *testing.T) {
	if _, _, err := Default().Resize(pngBytes(t, 10, 10), 5, "webp"); err == nil {
		t.Fatal("expected an error for unsupported webp format from the built-in processor")
	}
}

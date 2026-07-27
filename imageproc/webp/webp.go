// Package webp provides a buckt image Processor that also encodes WebP, using a
// pure-Go encoder (no cgo, no system libraries).
//
// It implements the exact method signature buckt's imageproc.Processor expects
// WITHOUT importing buckt, so the dependency stays one-way — the same rule the
// cloud storage backends follow. Wire it in with:
//
//	client, _ := buckt.New(buckt.Config{
//	    ImageProcessor: webp.New(),
//	    Derivatives: []buckt.DerivativeSpec{
//	        {Name: "thumbnail", MaxWidth: 200, Format: "webp"},
//	    },
//	})
package webp

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/HugoSmits86/nativewebp"
	"github.com/nfnt/resize"
	_ "golang.org/x/image/webp" // register the WebP decoder for image.Decode
)

// Processor resizes images and can encode JPEG, PNG, and WebP.
type Processor struct{}

// New returns a WebP-capable image processor.
func New() *Processor { return &Processor{} }

// Resize scales the image to at most maxWidth pixels wide (preserving aspect
// ratio, never upscaling) and encodes it. format may be "webp", "jpeg"/"jpg",
// "png", or "" to keep the decoded source format. Supported inputs are JPEG,
// PNG, GIF (first frame), and WebP.
func (Processor) Resize(data []byte, maxWidth uint, format string) ([]byte, string, error) {
	img, srcFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	if format == "" {
		format = srcFormat
	}

	w := uint(img.Bounds().Dx())
	if maxWidth == 0 || maxWidth > w {
		maxWidth = w
	}
	resized := resize.Resize(maxWidth, 0, img, resize.Lanczos3)

	var buf bytes.Buffer
	switch format {
	case "webp":
		if err := nativewebp.Encode(&buf, resized, nil); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/webp", nil
	case "png":
		if err := png.Encode(&buf, resized); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/png", nil
	case "jpeg", "jpg":
		if err := jpeg.Encode(&buf, resized, nil); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/jpeg", nil
	default:
		return nil, "", fmt.Errorf("webp processor: unsupported output format %q", format)
	}
}

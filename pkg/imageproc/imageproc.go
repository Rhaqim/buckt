// Package imageproc defines the pluggable image processor buckt uses to build
// image derivatives (thumbnails, etc.), plus a built-in pure-Go default.
//
// The Processor interface is stdlib-only by design: an alternative processor
// (e.g. one that also encodes WebP or AVIF) can implement it in its own module
// WITHOUT importing buckt, keeping the dependency one-way — the same rule the
// storage backends follow. Register one with buckt.WithImageProcessor.
package imageproc

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"github.com/nfnt/resize"
)

// Processor resizes and re-encodes images for derivative generation.
type Processor interface {
	// Resize scales data to at most maxWidth pixels wide (preserving aspect
	// ratio and never upscaling) and encodes it in format. An empty format means
	// "keep the source format". It returns the encoded bytes and their content
	// type, or an error for inputs/formats it does not support.
	Resize(data []byte, maxWidth uint, format string) (out []byte, contentType string, err error)
}

// Default returns buckt's built-in processor: pure-Go JPEG and PNG resizing with
// no cgo and no dependency beyond what buckt already ships. It does not encode
// WebP — plug in a Processor that does (see the webp module) for that.
func Default() Processor { return builtin{} }

type builtin struct{}

func (builtin) Resize(data []byte, maxWidth uint, format string) ([]byte, string, error) {
	img, srcFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	if format == "" {
		format = srcFormat
	}

	// Never upscale: cap the target width at the source width.
	w := uint(img.Bounds().Dx())
	if maxWidth == 0 || maxWidth > w {
		maxWidth = w
	}
	resized := resize.Resize(maxWidth, 0, img, resize.Lanczos3)

	var buf bytes.Buffer
	switch format {
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
		return nil, "", fmt.Errorf("imageproc: built-in processor does not support format %q (it supports jpeg and png); register a custom Processor via buckt.WithImageProcessor", format)
	}
}

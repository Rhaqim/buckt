package buckt

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makePNG builds a w×h PNG in memory for derivative tests.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func decodedWidth(t *testing.T, data []byte) int {
	t.Helper()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	require.NoError(t, err)
	return cfg.Width
}

func TestDerivatives_GenerateFetchAndScale(t *testing.T) {
	c := newTestClient(t, withNested(), withImageDerivatives(
		DerivativeSpec{Name: "thumbnail", MaxWidth: 100},
		DerivativeSpec{Name: "large", MaxWidth: 800}, // wider than source → no upscale
	))
	const user = "u1"

	id, err := c.UploadFile(user, "", "photo.png", "image/png", makePNG(t, 400, 300))
	require.NoError(t, err)

	// The app would call this from a file.uploaded handler / worker.
	require.NoError(t, c.GenerateDerivatives(id))

	thumb, ct, err := c.GetDerivative(id, "thumbnail")
	require.NoError(t, err)
	assert.Equal(t, "image/png", ct)
	assert.Equal(t, 100, decodedWidth(t, thumb), "thumbnail scaled to MaxWidth")

	large, _, err := c.GetDerivative(id, "large")
	require.NoError(t, err)
	assert.Equal(t, 400, decodedWidth(t, large), "never upscales past the source width")
}

func TestDerivatives_NonImageSkipped(t *testing.T) {
	c := newTestClient(t, withImageDerivatives(DerivativeSpec{Name: "thumbnail", MaxWidth: 100}))

	id, err := c.UploadFile("u1", "", "notes.txt", "text/plain", []byte("not an image"))
	require.NoError(t, err)

	// No-op for non-images; safe to call unconditionally.
	require.NoError(t, c.GenerateDerivatives(id))

	_, _, err = c.GetDerivative(id, "thumbnail")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestDerivatives_PurgeCleansUp(t *testing.T) {
	c := newTestClient(t, withNested(), withImageDerivatives(DerivativeSpec{Name: "thumbnail", MaxWidth: 100}))
	const user = "u1"

	id, err := c.UploadFile(user, "", "photo.png", "image/png", makePNG(t, 200, 200))
	require.NoError(t, err)
	require.NoError(t, c.GenerateDerivatives(id))

	// Trash then permanently delete → derivatives are removed from storage. We
	// fetch only after purge (a cache-cold read) so this asserts the object was
	// actually deleted, not just evicted from the backend read cache.
	_, err = c.DeleteFile(id)
	require.NoError(t, err)
	_, err = c.DeleteFilePermanently(id)
	require.NoError(t, err)

	_, _, err = c.GetDerivative(id, "thumbnail")
	assert.ErrorIs(t, err, ErrNotFound, "derivatives cleaned up on permanent delete")
}

func TestDerivatives_CountedInStorageBytes(t *testing.T) {
	c := newTestClient(t, withNested(), withImageDerivatives(
		DerivativeSpec{Name: "thumbnail", MaxWidth: 50},
	))
	const user = "u1"

	original := makePNG(t, 300, 300)
	id, err := c.UploadFile(user, "", "photo.png", "image/png", original)
	require.NoError(t, err)

	before, err := c.StorageBytes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(len(original)), before, "before generating, only the original counts")

	require.NoError(t, c.GenerateDerivatives(id))

	after, err := c.StorageBytes(context.Background())
	require.NoError(t, err)
	assert.Greater(t, after, before, "derivative bytes are now included in storage")

	// Fetch the derivative and confirm the delta equals its byte size.
	thumb, _, err := c.GetDerivative(id, "thumbnail")
	require.NoError(t, err)
	assert.Equal(t, before+int64(len(thumb)), after)
}

// fakeProcessor proves the WithImageProcessor plug-in path without pulling a
// real encoder into core: it echoes the requested format so the test can assert
// the processor was invoked and its output stored/retrieved.
type fakeProcessor struct{}

func (fakeProcessor) Resize(data []byte, maxWidth uint, format string) ([]byte, string, error) {
	return []byte("PROCESSED:" + format), "application/x-test", nil
}

func TestDerivatives_CustomProcessorPluggedIn(t *testing.T) {
	c := newTestClient(t, withNested(),
		withImageProcessor(fakeProcessor{}),
		withImageDerivatives(DerivativeSpec{Name: "thumb", MaxWidth: 100, Format: "webp"}),
	)

	id, err := c.UploadFile("u1", "", "photo.png", "image/png", makePNG(t, 200, 200))
	require.NoError(t, err)
	require.NoError(t, c.GenerateDerivatives(id))

	out, _, err := c.GetDerivative(id, "thumb")
	require.NoError(t, err)
	assert.Equal(t, "PROCESSED:webp", string(out), "custom processor is invoked with the spec's Format and its output is stored")
}

func TestDerivatives_NotConfiguredIsNoOp(t *testing.T) {
	c := newTestClient(t)
	id, err := c.UploadFile("u1", "", "photo.png", "image/png", makePNG(t, 120, 120))
	require.NoError(t, err)
	require.NoError(t, c.GenerateDerivatives(id), "no-op when no variants configured")
}

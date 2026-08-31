package buckt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPresignedURL_UnsupportedOnLocal verifies that presigning is reported as
// unsupported (not a crash) on the local filesystem backend, which can't mint
// direct URLs. Cloud presigning is covered in cloud/aws (no network needed).
func TestPresignedURL_UnsupportedOnLocal(t *testing.T) {
	c := newTestClient(t) // local backend
	fileID, err := c.UploadFile("u1", "", "a.txt", "text/plain", []byte("x"))
	require.NoError(t, err)

	_, err = c.PresignedURL(fileID, 15*time.Minute)
	assert.ErrorIs(t, err, ErrUnsupported, "local backend cannot presign")

	_, err = c.PresignedDerivativeURL(fileID, "thumbnail", 15*time.Minute)
	assert.ErrorIs(t, err, ErrUnsupported)
}

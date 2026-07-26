package buckt

import (
	"errors"
	"testing"

	"github.com/Rhaqim/buckt/pkg/buckterr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientSentinelErrors(t *testing.T) {
	c := newTestClient(t, withMaxFileSize(1024))
	const user = "u1"

	t.Run("invalid id", func(t *testing.T) {
		_, err := c.GetFile("not-a-uuid")
		assert.ErrorIs(t, err, ErrInvalidID)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := c.GetFile(uuid.NewString()) // well-formed but absent
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("invalid file name", func(t *testing.T) {
		_, err := c.UploadFile(user, "", "../evil.txt", "text/plain", []byte("hi"))
		assert.ErrorIs(t, err, ErrInvalidName)
	})

	t.Run("file too large", func(t *testing.T) {
		_, err := c.UploadFile(user, "", "big.bin", "application/octet-stream", make([]byte, 2048))
		assert.ErrorIs(t, err, ErrFileTooLarge)
	})

	t.Run("invalid folder name", func(t *testing.T) {
		_, err := c.NewFolder(user, "", "bad/name", "")
		assert.ErrorIs(t, err, ErrInvalidName)
	})

	t.Run("already exists", func(t *testing.T) {
		_, err := c.UploadFile(user, "", "dup.txt", "text/plain", []byte("one"))
		require.NoError(t, err)
		_, err = c.UploadFile(user, "", "dup.txt", "text/plain", []byte("two"))
		assert.ErrorIs(t, err, ErrAlreadyExists)
	})

	t.Run("root re-export equals buckterr value", func(t *testing.T) {
		// The same underlying value, so errors.Is works via either package.
		assert.True(t, errors.Is(ErrNotFound, buckterr.ErrNotFound))
		assert.Same(t, ErrInvalidName, buckterr.ErrInvalidName)
	})
}

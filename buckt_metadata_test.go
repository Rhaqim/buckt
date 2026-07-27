package buckt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileMetadata_SetGetRoundTrip(t *testing.T) {
	c := newTestClient(t)
	const user = "u1"

	fileID, err := c.UploadFile(user, "", "dress.jpg", "image/jpeg", []byte("bytes"))
	require.NoError(t, err)

	// No metadata initially.
	meta, err := c.GetFileMetadata(fileID)
	require.NoError(t, err)
	assert.Empty(t, meta)

	// Set some (e.g. the "attachments" resource link + AI tags).
	in := map[string]string{
		"resource_type": "order",
		"resource_id":   "42",
		"garment":       "dress",
		"color":         "red",
	}
	require.NoError(t, c.SetFileMetadata(fileID, in))

	got, err := c.GetFileMetadata(fileID)
	require.NoError(t, err)
	assert.Equal(t, in, got)

	// It also travels with the file on GetFile.
	f, err := c.GetFile(fileID)
	require.NoError(t, err)
	assert.Equal(t, in, f.Metadata)
}

func TestFileMetadata_SetReplaces(t *testing.T) {
	c := newTestClient(t)

	fileID, err := c.UploadFile("u1", "", "a.txt", "text/plain", []byte("x"))
	require.NoError(t, err)

	require.NoError(t, c.SetFileMetadata(fileID, map[string]string{"a": "1", "b": "2"}))
	require.NoError(t, c.SetFileMetadata(fileID, map[string]string{"c": "3"}))

	got, err := c.GetFileMetadata(fileID)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"c": "3"}, got, "SetFileMetadata replaces the whole map")
}

func TestFileMetadata_UnknownFile(t *testing.T) {
	c := newTestClient(t)
	err := c.SetFileMetadata("00000000-0000-0000-0000-000000000000", map[string]string{"a": "1"})
	assert.ErrorIs(t, err, ErrNotFound)
}

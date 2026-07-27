package buckt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDedup_SameContentSameFolderReturnsExisting(t *testing.T) {
	c := newTestClient(t, withNested(), withDedup())
	const user = "u1"

	docsID, err := c.NewFolder(user, "", "Docs", "")
	require.NoError(t, err)

	content := []byte("identical bytes from whatsapp")

	// First upload stores the file.
	id1, err := c.UploadFile(user, docsID, "dress.jpg", "image/jpeg", content)
	require.NoError(t, err)

	// Second upload of the SAME bytes into the SAME folder (different name)
	// resolves to the existing file — no new blob stored.
	id2, err := c.UploadFile(user, docsID, "dress (1).jpg", "image/jpeg", content)
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "identical content in the same folder dedups to one file")

	// A third, differently-named copy also dedups.
	id3, err := c.UploadFile(user, docsID, "IMG_1929.jpg", "image/jpeg", content)
	require.NoError(t, err)
	assert.Equal(t, id1, id3)

	// Only one file physically lives in the folder.
	docs, err := c.GetFolderWithContent(user, docsID)
	require.NoError(t, err)
	assert.Len(t, docs.Files, 1, "duplicates were not stored")

	// Storage reflects a single blob, not three.
	total, err := c.StorageBytes(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), total)
}

func TestDedup_DifferentContentNotDeduped(t *testing.T) {
	c := newTestClient(t, withDedup())
	const user = "u1"

	id1, err := c.UploadFile(user, "", "a.txt", "text/plain", []byte("one"))
	require.NoError(t, err)
	id2, err := c.UploadFile(user, "", "b.txt", "text/plain", []byte("two"))
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2, "distinct content is stored separately")
}

func TestDedup_DisabledByDefault(t *testing.T) {
	c := newTestClient(t) // dedup off
	const user = "u1"

	content := []byte("same bytes")
	id1, err := c.UploadFile(user, "", "a.txt", "text/plain", content)
	require.NoError(t, err)
	// Without dedup, a different name is a distinct file even with identical bytes.
	id2, err := c.UploadFile(user, "", "b.txt", "text/plain", content)
	require.NoError(t, err)
	assert.NotEqual(t, id1, id2, "dedup must be opt-in")
}

func TestDedup_ScopedPerFolder(t *testing.T) {
	c := newTestClient(t, withNested(), withDedup())
	const user = "u1"

	a, err := c.NewFolder(user, "", "A", "")
	require.NoError(t, err)
	b, err := c.NewFolder(user, "", "B", "")
	require.NoError(t, err)

	content := []byte("shared bytes")
	idA, err := c.UploadFile(user, a, "x.bin", "application/octet-stream", content)
	require.NoError(t, err)
	// Same content in a DIFFERENT folder is a separate file — dedup is scoped to
	// the target folder so it composes with nested-namespace paths.
	idB, err := c.UploadFile(user, b, "x.bin", "application/octet-stream", content)
	require.NoError(t, err)
	assert.NotEqual(t, idA, idB)
}

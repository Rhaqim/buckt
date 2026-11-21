package buckt

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rhaqim/buckt/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// headerMemBackend is a memBackend that also supports signed response-header
// overrides, recording the values PresignedURL passes so tests can assert them.
type headerMemBackend struct {
	*memBackend
	gotType, gotDisposition, gotCache string
}

func (h *headerMemBackend) PresignGetURLWithHeaders(_ context.Context, key string, _ time.Duration, contentType, contentDisposition, cacheControl string) (string, error) {
	h.gotType, h.gotDisposition, h.gotCache = contentType, contentDisposition, cacheControl
	return "memget-h://" + key, nil
}

func headerPresignClient(t *testing.T, opts ...ConfigFunc) (*Client, *headerMemBackend) {
	t.Helper()
	dir := t.TempDir()
	sqlDB, err := sql.Open("sqlite3", filepath.Join(dir, "b.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	target := &headerMemBackend{memBackend: newMemBackend()}
	c, err := New(Config{
		DB:       DBConfig{Driver: SQLite, Database: sqlDB},
		MediaDir: filepath.Join(dir, "media"),
		Log:      LogConfig{Silence: true},
		Backend:  BackendConfig{Source: target},
	}, opts...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	return c, target
}

// TestPresignedURL_SignsResponseHeaders: the recorded content type, an inline
// disposition for inline-safe media, and a ttl-derived Cache-Control are passed.
func TestPresignedURL_SignsResponseHeaders(t *testing.T) {
	c, be := headerPresignClient(t)
	id, err := c.UploadFile("u1", "", "voice.m4a", "audio/mp4", []byte("fake-audio"))
	require.NoError(t, err)

	url, err := c.PresignedURL(id, 15*time.Minute)
	require.NoError(t, err)
	assert.Contains(t, url, "memget-h://")
	assert.Equal(t, "audio/mp4", be.gotType)
	assert.Equal(t, "inline; filename*=UTF-8''voice.m4a", be.gotDisposition)
	assert.Equal(t, "private, max-age=900", be.gotCache)
}

// TestPresignedURL_UnsafeTypeIsAttachment: HTML must download, never render.
func TestPresignedURL_UnsafeTypeIsAttachment(t *testing.T) {
	c, be := headerPresignClient(t)
	id, err := c.UploadFile("u1", "", "page.html", "text/html", []byte("<script>alert(1)</script>"))
	require.NoError(t, err)

	_, err = c.PresignedURL(id, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "text/html", be.gotType)
	assert.Equal(t, "attachment; filename*=UTF-8''page.html", be.gotDisposition)
}

func TestPresignedURL_CacheControlOption(t *testing.T) {
	c, be := headerPresignClient(t, WithPresignCacheControl("public, max-age=86400"))
	id, err := c.UploadFile("u1", "", "a.png", "image/png", []byte("\x89PNG\r\n\x1a\n"))
	require.NoError(t, err)

	_, err = c.PresignedURL(id, time.Minute)
	require.NoError(t, err)
	assert.Equal(t, "public, max-age=86400", be.gotCache)
}

// TestPresignedURL_HeadersSurviveMetrics: the metering wrapper must forward the
// header-override capability, not silently drop to a plain URL.
func TestPresignedURL_HeadersSurviveMetrics(t *testing.T) {
	c, be := headerPresignClient(t, WithMetrics(metrics.NewCollector()))
	id, err := c.UploadFile("u1", "", "voice.m4a", "audio/mp4", []byte("x"))
	require.NoError(t, err)

	url, err := c.PresignedURL(id, time.Minute)
	require.NoError(t, err)
	assert.Contains(t, url, "memget-h://")
	assert.Equal(t, "audio/mp4", be.gotType)
}

// TestPresignedURL_FallbackWithoutHeaderSupport: a backend with only plain
// presigning (memBackend) still gets a URL, with or without the metering wrapper.
func TestPresignedURL_FallbackWithoutHeaderSupport(t *testing.T) {
	for _, withMetrics := range []bool{false, true} {
		t.Run(fmt.Sprintf("metrics=%v", withMetrics), func(t *testing.T) {
			dir := t.TempDir()
			sqlDB, err := sql.Open("sqlite3", filepath.Join(dir, "b.db"))
			require.NoError(t, err)
			t.Cleanup(func() { _ = sqlDB.Close() })

			var opts []ConfigFunc
			if withMetrics {
				opts = append(opts, WithMetrics(metrics.NewCollector()))
			}
			c, err := New(Config{
				DB:       DBConfig{Driver: SQLite, Database: sqlDB},
				MediaDir: filepath.Join(dir, "media"),
				Log:      LogConfig{Silence: true},
				Backend:  BackendConfig{Source: newMemBackend()},
			}, opts...)
			require.NoError(t, err)
			t.Cleanup(func() { _ = c.Close() })

			id, err := c.UploadFile("u1", "", "a.txt", "text/plain", []byte("x"))
			require.NoError(t, err)
			url, err := c.PresignedURL(id, time.Minute)
			require.NoError(t, err)
			assert.Contains(t, url, "memget://")
		})
	}
}

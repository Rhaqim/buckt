package gcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GCPBackend struct {
	client     *storage.Client
	bucketName string
}

func NewBackend(conf Config) (*GCPBackend, error) {
	err := conf.Validate()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	var opts []option.ClientOption
	if conf.Endpoint != "" {
		// Emulator / private endpoint: skip auth and target the given API URL.
		opts = append(opts, option.WithEndpoint(conf.Endpoint), option.WithoutAuthentication())
	} else {
		// WithCredentialsFile is deprecated upstream but remains the supported way
		// to pass a service-account key file path, which is this backend's
		// configured auth model. Migrating to ADC / WithCredentialsJSON is a
		// separate change.
		opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, conf.CredentialsFile)) //nolint:staticcheck // intentional: key-file auth is the configured model
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP client: %w", err)
	}

	return &GCPBackend{
		client:     client,
		bucketName: conf.Bucket,
	}, nil
}

func (g *GCPBackend) Name() string {
	return "gcp"
}

// altKey returns the leading-slash-stripped form of an object name and whether it
// differs. buckt's nested-namespace paths carry a leading slash (/user/...), but
// a local -> GCS migration copies objects under the local backend's relative
// List() names (no leading slash). Reads and deletes try both so objects written
// by either path resolve. See the aws backend for the full rationale.
func altKey(path string) (string, bool) {
	alt := strings.TrimPrefix(path, "/")
	return alt, alt != path
}

// isNotFound reports whether err is a "object missing" error.
func isNotFound(err error) bool {
	return errors.Is(err, storage.ErrObjectNotExist)
}

// Ping verifies connectivity to the GCS bucket by checking its attributes.
func (g *GCPBackend) Ping(ctx context.Context) error {
	_, err := g.client.Bucket(g.bucketName).Attrs(ctx)
	if err != nil {
		return fmt.Errorf("GCS connectivity check failed for bucket %q: %w", g.bucketName, err)
	}
	return nil
}

func (g *GCPBackend) Put(ctx context.Context, path string, data []byte) error {
	w := g.client.Bucket(g.bucketName).Object(path).NewWriter(ctx)
	if _, err := w.Write(data); err != nil {
		// Cancel the upload by closing the writer — GCS will discard the incomplete object
		_ = w.Close()
		return fmt.Errorf("failed to write object: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}
	return nil
}

func (g *GCPBackend) Get(ctx context.Context, path string) ([]byte, error) {
	data, err := g.getObject(ctx, path)
	if err != nil && isNotFound(err) {
		if alt, ok := altKey(path); ok {
			if d, altErr := g.getObject(ctx, alt); altErr == nil {
				return d, nil
			}
		}
	}
	return data, err
}

func (g *GCPBackend) getObject(ctx context.Context, key string) ([]byte, error) {
	r, err := g.client.Bucket(g.bucketName).Object(key).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}
	return buf.Bytes(), nil
}

func (g *GCPBackend) List(ctx context.Context, prefix string) ([]string, error) {
	var results []string
	it := g.client.Bucket(g.bucketName).Objects(ctx, &storage.Query{Prefix: prefix})

	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}
		results = append(results, obj.Name)
	}
	return results, nil
}

func (g *GCPBackend) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	r, err := g.streamObject(ctx, path)
	if err != nil && isNotFound(err) {
		if alt, ok := altKey(path); ok {
			if rc, altErr := g.streamObject(ctx, alt); altErr == nil {
				return rc, nil
			}
		}
	}
	return r, err
}

func (g *GCPBackend) streamObject(ctx context.Context, key string) (io.ReadCloser, error) {
	r, err := g.client.Bucket(g.bucketName).Object(key).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil // caller must close
}

// Delete removes the object under both key forms so a migrated copy (relative
// key) is cleaned up too. A missing object is treated as already-deleted.
func (g *GCPBackend) Delete(ctx context.Context, path string) error {
	err := g.deleteKey(ctx, path)
	if alt, ok := altKey(path); ok {
		if altErr := g.deleteKey(ctx, alt); altErr != nil && err == nil {
			err = altErr
		}
	}
	return err
}

func (g *GCPBackend) deleteKey(ctx context.Context, key string) error {
	if err := g.client.Bucket(g.bucketName).Object(key).Delete(ctx); err != nil {
		if isNotFound(err) {
			return nil // idempotent: already gone
		}
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (g *GCPBackend) Exists(ctx context.Context, path string) (bool, error) {
	exists, err := g.existsKey(ctx, path)
	if err == nil && !exists {
		if alt, ok := altKey(path); ok {
			return g.existsKey(ctx, alt)
		}
	}
	return exists, err
}

func (g *GCPBackend) existsKey(ctx context.Context, key string) (bool, error) {
	_, err := g.client.Bucket(g.bucketName).Object(key).Attrs(ctx)
	if isNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return true, nil
}

func (g *GCPBackend) Stat(ctx context.Context, path string) (*FileInfo, error) {
	attrs, err := g.client.Bucket(g.bucketName).Object(path).Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}
	return &FileInfo{
		Size:         attrs.Size,
		LastModified: attrs.Updated,
		ETag:         attrs.Etag,
		ContentType:  attrs.ContentType,
	}, nil
}

func (g *GCPBackend) DeleteFolder(ctx context.Context, prefix string) error {
	if err := g.deletePrefix(ctx, prefix+"/"); err != nil {
		return err
	}
	// Also clear objects a migration wrote under the leading-slash-stripped key.
	if alt, ok := altKey(prefix); ok {
		if err := g.deletePrefix(ctx, alt+"/"); err != nil {
			return err
		}
	}
	return nil
}

func (g *GCPBackend) deletePrefix(ctx context.Context, prefix string) error {
	it := g.client.Bucket(g.bucketName).Objects(ctx, &storage.Query{Prefix: prefix})

	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}
		if delErr := g.deleteKey(ctx, obj.Name); delErr != nil {
			return fmt.Errorf("failed to delete object %s: %w", obj.Name, delErr)
		}
	}
	return nil
}

func (g *GCPBackend) Move(ctx context.Context, oldPath, newPath string) error {
	// The source may live under either key form (direct write vs. migrated copy).
	srcKey := oldPath
	if alt, ok := altKey(oldPath); ok {
		if exists, _ := g.existsKey(ctx, oldPath); !exists {
			srcKey = alt
		}
	}

	src := g.client.Bucket(g.bucketName).Object(srcKey)
	dst := g.client.Bucket(g.bucketName).Object(newPath)

	// Copy
	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	// Delete old (both key forms)
	if err := g.Delete(ctx, oldPath); err != nil {
		return fmt.Errorf("failed to delete old object: %w", err)
	}
	return nil
}

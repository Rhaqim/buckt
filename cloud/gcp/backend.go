package gcp

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GCPBackend struct {
	client     *storage.Client
	bucketName string
}

func NewBackend(conf GCPConfig) (*GCPBackend, error) {
	err := conf.Validate()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(conf.CredentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP client: %w", err)
	}

	return &GCPBackend{
		client:     client,
		bucketName: conf.Bucket,
	}, nil
}

func (g *GCPBackend) Put(path string, data []byte) error {
	ctx := context.TODO()
	w := g.client.Bucket(g.bucketName).Object(path).NewWriter(ctx)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write object: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}
	return nil
}

func (g *GCPBackend) Get(path string) ([]byte, error) {
	ctx := context.TODO()
	r, err := g.client.Bucket(g.bucketName).Object(path).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open reader: %w", err)
	}
	defer r.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}
	return buf.Bytes(), nil
}

func (g *GCPBackend) Stream(path string) (io.ReadCloser, error) {
	ctx := context.TODO()
	r, err := g.client.Bucket(g.bucketName).Object(path).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return r, nil // caller must close
}

func (g *GCPBackend) Delete(path string) error {
	ctx := context.TODO()
	if err := g.client.Bucket(g.bucketName).Object(path).Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

func (g *GCPBackend) Exists(path string) (bool, error) {
	ctx := context.TODO()
	_, err := g.client.Bucket(g.bucketName).Object(path).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return true, nil
}

func (g *GCPBackend) Stat(path string) (*FileInfo, error) {
	ctx := context.TODO()
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

func (g *GCPBackend) DeleteFolder(prefix string) error {
	ctx := context.TODO()
	it := g.client.Bucket(g.bucketName).Objects(ctx, &storage.Query{Prefix: prefix + "/"})

	for {
		obj, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}
		if delErr := g.client.Bucket(g.bucketName).Object(obj.Name).Delete(ctx); delErr != nil {
			return fmt.Errorf("failed to delete object %s: %w", obj.Name, delErr)
		}
	}
	return nil
}

func (g *GCPBackend) Move(oldPath, newPath string) error {
	ctx := context.TODO()
	src := g.client.Bucket(g.bucketName).Object(oldPath)
	dst := g.client.Bucket(g.bucketName).Object(newPath)

	// Copy
	if _, err := dst.CopierFrom(src).Run(ctx); err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	// Delete old
	if err := src.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete old object: %w", err)
	}
	return nil
}

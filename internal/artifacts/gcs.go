package artifacts

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCSStore stores artifacts in Google Cloud Storage.
type GCSStore struct {
	client *storage.Client
	bucket string
	prefix string
}

// NewGCSStore creates a GCSStore using Application Default Credentials.
func NewGCSStore(ctx context.Context, bucket, prefix string) (*GCSStore, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs client: %w", err)
	}
	return &GCSStore{client: client, bucket: bucket, prefix: prefix}, nil
}

func (g *GCSStore) object(runID, path string) string {
	if g.prefix != "" {
		return g.prefix + "/" + runID + "/" + path
	}
	return runID + "/" + path
}

func (g *GCSStore) Upload(ctx context.Context, runID, path string, r io.Reader) error {
	wc := g.client.Bucket(g.bucket).Object(g.object(runID, path)).NewWriter(ctx)
	if _, err := io.Copy(wc, r); err != nil {
		_ = wc.Close()
		return fmt.Errorf("gcs upload %s/%s: %w", runID, path, err)
	}
	return wc.Close()
}

func (g *GCSStore) Download(ctx context.Context, runID, path string) (io.ReadCloser, error) {
	rc, err := g.client.Bucket(g.bucket).Object(g.object(runID, path)).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcs download %s/%s: %w", runID, path, err)
	}
	return rc, nil
}

func (g *GCSStore) List(ctx context.Context, runID string) ([]string, error) {
	prefix := g.object(runID, "")
	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{Prefix: prefix})
	var paths []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gcs list %s: %w", runID, err)
		}
		paths = append(paths, attrs.Name[len(prefix):])
	}
	return paths, nil
}

func (g *GCSStore) Close() error { return g.client.Close() }

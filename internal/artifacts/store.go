// Package artifacts defines the ArtifactStore interface and its implementations.
package artifacts

import (
	"context"
	"io"
)

// ArtifactStore persists and retrieves job artifacts by run ID and path.
type ArtifactStore interface {
	Upload(ctx context.Context, runID, path string, r io.Reader) error
	Download(ctx context.Context, runID, path string) (io.ReadCloser, error)
	List(ctx context.Context, runID string) ([]string, error)
}

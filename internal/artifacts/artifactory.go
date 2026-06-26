package artifacts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ArtifactoryStore stores artifacts in a JFrog Artifactory generic repository.
type ArtifactoryStore struct {
	baseURL string // e.g. https://acme.jfrog.io/artifactory/brazier-artifacts
	token   string
	client  *http.Client
}

// NewArtifactoryStore returns an ArtifactoryStore targeting baseURL, authenticated with token.
func NewArtifactoryStore(baseURL, token string) *ArtifactoryStore {
	return &ArtifactoryStore{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  &http.Client{},
	}
}

func (a *ArtifactoryStore) artifactURL(runID, path string) string {
	return fmt.Sprintf("%s/%s/%s", a.baseURL, runID, path)
}

func (a *ArtifactoryStore) Upload(ctx context.Context, runID, path string, r io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, a.artifactURL(runID, path), r)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("artifactory upload: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("artifactory upload: status %d", resp.StatusCode)
	}
	return nil
}

func (a *ArtifactoryStore) Download(ctx context.Context, runID, path string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.artifactURL(runID, path), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+a.token)
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("artifactory download: %w", err)
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("artifactory download: status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func (a *ArtifactoryStore) List(_ context.Context, _ string) ([]string, error) {
	return nil, fmt.Errorf("artifactory list: not implemented")
}

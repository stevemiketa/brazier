package artifacts

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStore writes artifacts to the local filesystem under baseDir.
type LocalStore struct {
	baseDir string
}

// NewLocalStore returns a LocalStore rooted at baseDir.
func NewLocalStore(baseDir string) *LocalStore {
	return &LocalStore{baseDir: baseDir}
}

func (s *LocalStore) artifactPath(runID, path string) string {
	return filepath.Join(s.baseDir, runID, filepath.Clean(path))
}

func (s *LocalStore) Upload(_ context.Context, runID, path string, r io.Reader) error {
	dest := s.artifactPath(runID, path)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create artifact: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write artifact: %w", err)
	}
	return nil
}

func (s *LocalStore) Download(_ context.Context, runID, path string) (io.ReadCloser, error) {
	src := s.artifactPath(runID, path)
	f, err := os.Open(src)
	if err != nil {
		return nil, fmt.Errorf("open artifact: %w", err)
	}
	return f, nil
}

func (s *LocalStore) List(_ context.Context, runID string) ([]string, error) {
	root := filepath.Join(s.baseDir, runID)
	var paths []string
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		paths = append(paths, strings.ReplaceAll(rel, string(filepath.Separator), "/"))
		return nil
	})
	if os.IsNotExist(err) {
		return nil, nil
	}
	return paths, err
}

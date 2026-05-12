package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

type FilesystemClient struct {
	baseDir string
}

func NewClient(baseDir string) (*FilesystemClient, error) {
	// Clean the path and ensure the directory exists with correct permissions
	cleanDir := filepath.Clean(baseDir)
	if err := os.MkdirAll(cleanDir, 0644); err != nil {
		return nil, fmt.Errorf("failed to create base directory %s: %w", cleanDir, err)
	}

	return &FilesystemClient{
		baseDir: cleanDir,
	}, nil
}

func (c *FilesystemClient) Write() {

}

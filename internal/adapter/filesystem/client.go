package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

type Client struct {
	baseDir string
}

func NewClient(baseDir string) (*Client, error) {
	// Clean the path and ensure the directory exists with correct permissions
	cleanDir := filepath.Clean(baseDir)
	if err := os.MkdirAll(cleanDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory %s: %w", cleanDir, err)
	}

	return &Client{
		baseDir: cleanDir,
	}, nil
}

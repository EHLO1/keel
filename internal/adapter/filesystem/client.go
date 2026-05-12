package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type FilesystemClient struct {
	baseDir string
}

// make a map of paths to initialize the clients!

func NewClient(baseDir string) (*FilesystemClient, error) {
	// Clean the provided baseDir, create if it doesn't exist, fix permissions.
	cleanDir := filepath.Clean(baseDir)
	if err := os.MkdirAll(cleanDir, 0644); err != nil {
		return nil, fmt.Errorf("failed to create base directory %s: %w", cleanDir, err)
	}

	return &FilesystemClient{
		baseDir: cleanDir,
	}, nil
}

func (c *FilesystemClient) SetMaintenanceMode(mm bool) error {
	mfe, err := maintFileExists(c)
	if err != nil {
		return err
	}

	switch {
	case !mm && !mfe:
		return fmt.Errorf("maintenance mode is not enabled: %s", err)

	case !mm && mfe:
		_, err := os.Create(c.baseDir + "/maintenance_mode")
		if err != nil {
			return fmt.Errorf("failed to enable maintenance mode: %s", err)
		}

	case mm && mfe:
		return fmt.Errorf("maintenance mode is already enabled: %s", err)

	case mm && !mfe:
		err := os.Remove(c.baseDir + "/maintenance_mode")
		if err != nil {
			return fmt.Errorf("failed to disable maintenance mode: %s", err)
		}

	default:
		panic(fmt.Errorf("maintenance mode unknown: %s", err))
	}
	return nil
}

func maintFileExists(c *FilesystemClient) (bool, error) {
	_, err := os.Stat(c.baseDir + "/maintenance_mode")
	if !errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	return false, err
}

func (c *FilesystemClient) GetStandbySignal() (bool, error) {
	sse, err := standbySignalExists(c)
	if err != nil {
		return false, err
	}

	switch {
	case sse:
		return true, nil

	case !sse:
		return false, nil

	default:
		panic(fmt.Errorf("standby signal status unknown: %s", err))
	}
}

func standbySignalExists(dv string) (bool, error) {
	_, err := os.Stat(dv + "/maintenance_mode")
	if !errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	return false, err
}

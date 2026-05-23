package filesystem

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func mkdirWithPerms(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0744); err != nil {
		return err
	}
	return nil
}

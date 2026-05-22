package filesystem

import (
	"os"
	"path/filepath"
)

type StateFile struct {
	file string
}

func NewStateFile(path string, fileName string) *StateFile {
	return &StateFile{
		file: filepath.Join(path, fileName),
	}
}

func (s *StateFile) SetHealthy() error {
	// Ensure the parent directory exists
	dir := filepath.Dir(s.file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(s.file, []byte("OK\n"), 0644)
}

func (s *StateFile) SetUnhealthy() error {
	err := os.Remove(s.file)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

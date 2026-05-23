package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type StateFile struct {
	file string
	log  *slog.Logger
}

func NewStateFile(path string, fileName string, log *slog.Logger) *StateFile {
	sf := &StateFile{
		file: filepath.Join(path, fileName),
		log:  log,
	}

	if err := mkdirWithPerms(path); err != nil {
		log.Error("failed to create directory", "error", err)
		return sf
	}

	return sf
}

func (s *StateFile) Observe() (string, error) {
	f, err := os.ReadFile(s.file)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("state file does not exist: %w", err)
		}
		return "", err
	}
	return strings.TrimSpace(string(f)), nil
}

func (s *StateFile) SetHealthy() error {

	return os.WriteFile(s.file, []byte("OK\n"), 0644)
}

func (s *StateFile) SetUnhealthy() error {
	err := os.Remove(s.file)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

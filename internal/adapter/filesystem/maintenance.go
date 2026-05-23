package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

type MaintenanceFlag struct {
	file string
	log  *slog.Logger
}

func NewMaintenanceFlag(path string, fileName string, log *slog.Logger) *MaintenanceFlag {
	mf := &MaintenanceFlag{
		file: filepath.Join(path, fileName),
		log:  log,
	}

	if err := mkdirWithPerms(path); err != nil {
		log.Error("failed to create directory", "error", err)
		return mf
	}

	return mf
}

var ErrAlreadyEnabled = errors.New("maintenance mode is already enabled")
var ErrNotEnabled = errors.New("maintenance mode is not enabled")

func (m *MaintenanceFlag) Observe() (bool, error) {
	return exists(m.file)
}

func (m *MaintenanceFlag) Enable() error {
	f, err := os.OpenFile(m.file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if errors.Is(err, fs.ErrExist) {
		return ErrAlreadyEnabled
	}
	if err != nil {
		return fmt.Errorf("enabling maintenance mode: %w", err)
	}
	return f.Close()
}

func (m *MaintenanceFlag) Disable() error {
	err := os.Remove(m.file)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotEnabled
	}
	if err != nil {
		return fmt.Errorf("disabling maintenance mode: %w", err)
	}
	return nil
}

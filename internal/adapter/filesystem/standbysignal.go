package filesystem

import (
	"os"
	"path/filepath"
)

type StandbySignal struct {
	file string
}

func NewStandbySignal(fileName string) *StandbySignal {
	return &StandbySignal{
		file: fileName,
	}
}

func (s *StandbySignal) Observe(path string) (bool, error) {
	return exists(filepath.Join(path, s.file))
}

func (s *StandbySignal) SetStandby(path string) error {
	return os.WriteFile(filepath.Join(path, s.file), []byte(""), 0644)
}

func (s *StandbySignal) RemoveStandby(path string) error {
	err := os.Remove(filepath.Join(path, s.file))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

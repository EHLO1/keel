package filesystem

import (
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

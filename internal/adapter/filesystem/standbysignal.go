package filesystem

import "path/filepath"

type StandbySignal struct {
	file string
}

func NewStandbySignal(path string, fileName string) *StandbySignal {
	file := filepath.Clean(path) + fileName

	return &StandbySignal{
		file: file,
	}
}

func (s *StandbySignal) Present() (bool, error) {
	return exists(s.file)
}

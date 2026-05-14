package filesystem

type StandbySignal struct {
	file string
}

func NewStandbySignal(fileName string) *StandbySignal {
	file := fileName

	return &StandbySignal{
		file: file,
	}
}

func (s *StandbySignal) Present(path string) (bool, error) {
	return exists(path + "/" + s.file)
}

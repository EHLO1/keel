package policy

import "github.com/EHLO1/keel/internal/state"

type Evaluator struct{}

func NewEvaluator() (*Evaluator, error) {
	return &Evaluator{}, nil
}

func (e *Evaluator) Evaluate(snapshot *state.Snapshot) *DesiredState {
	return &DesiredState{}
}

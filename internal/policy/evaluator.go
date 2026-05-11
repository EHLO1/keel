package policy

import "github.com/EHLO1/keel/internal/types"

type PolicyEvaluator struct{}

func NewEvaluator() (*PolicyEvaluator, error) {
	return &PolicyEvaluator{}, nil
}

func (e *PolicyEvaluator) Evaluate(snapshot *types.Snapshot) *types.DesiredState {
	return &types.DesiredState{}
}

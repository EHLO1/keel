package policy

import "github.com/EHLO1/keel/internal/types"

type Evaluator interface {
	Evaluate(snapshot *types.Snapshot) *types.DesiredState
}

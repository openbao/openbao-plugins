package scheduler

import (
	"context"
	"fmt"
)

type repeatingProcess struct {
	ch       chan struct{}
	delegate Process
}

func (rp *repeatingProcess) Description() string {
	return fmt.Sprintf("(Repeating) %s", rp.delegate.Description())
}

func (rp *repeatingProcess) Run(ctx context.Context) error {
	defer close(rp.ch)

	return rp.delegate.Run(ctx)
}

// RepeatingDescriptor schedules a given process repeatedly. It does not allow
// process executions to overlap; i.e., an execution of a process immediately
// follows the completion of the prior execution.
type RepeatingDescriptor struct {
	process Process
}

var _ Descriptor = &RepeatingDescriptor{}

func (rd *RepeatingDescriptor) runOnce(ctx context.Context, pc chan<- Process) bool {
	ch := make(chan struct{})

	select {
	case <-ctx.Done():
		return false
	case pc <- &repeatingProcess{ch: ch, delegate: rd.process}:
	}

	select {
	case <-ctx.Done():
		return false
	case <-ch:
	}

	return true
}

// Run starts scheduling this descriptor's process. It terminates only when the
// context is done.
func (rd *RepeatingDescriptor) Run(ctx context.Context, pc chan<- Process) error {
	for {
		if !rd.runOnce(ctx, pc) {
			break
		}
	}

	return nil
}

// NewRepeatingDescriptor creates a new repeating descriptor that emits the
// given process.
func NewRepeatingDescriptor(process Process) *RepeatingDescriptor {
	return &RepeatingDescriptor{
		process: process,
	}
}

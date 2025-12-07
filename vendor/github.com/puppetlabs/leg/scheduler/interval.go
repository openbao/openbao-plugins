package scheduler

import (
	"context"
	"time"
)

// IntervalDescriptor schedules a given process many times with the same time
// duration between subsequent runs.
//
// The interval refers specifically to the time between executions; it is
// relative to the time the prior execution ended, not the time said execution
// started.
//
// For example, starting from midnight (00:00:00), given an interval of 60
// seconds, and a process that takes 20 seconds to execute, the second execution
// will occur around 00:01:20.
type IntervalDescriptor struct {
	interval time.Duration
	process  Process
}

var _ Descriptor = &IntervalDescriptor{}

func (id *IntervalDescriptor) runOnce(ctx context.Context, pc chan<- Process) (bool, error) {
	select {
	case <-ctx.Done():
		return false, nil
	case pc <- id.process:
	}

	t := time.NewTimer(id.interval)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return false, nil
	case <-t.C:
	}

	return true, nil
}

// Run starts scheduling this descriptor's process to the given channel. It will
// terminate only when the context terminates.
func (id *IntervalDescriptor) Run(ctx context.Context, pc chan<- Process) error {
	for {
		if cont, err := id.runOnce(ctx, pc); err != nil {
			return err
		} else if !cont {
			break
		}
	}

	return nil
}

// NewIntervalDescriptor creates a new descriptor that repeats the given process
// according to the given interval.
func NewIntervalDescriptor(interval time.Duration, process Process) *IntervalDescriptor {
	return &IntervalDescriptor{
		interval: interval,
		process:  process,
	}
}

package scheduler

import (
	"context"
)

// ImmediateDescriptor schedules a given process exactly once and then
// terminates.
type ImmediateDescriptor struct {
	process Process
}

var _ Descriptor = &ImmediateDescriptor{}

// Run schedules the process specified in this descriptor.
func (id *ImmediateDescriptor) Run(ctx context.Context, pc chan<- Process) error {
	select {
	case <-ctx.Done():
	case pc <- id.process:
	}

	return nil
}

// NewImmediateDescriptor creates an immediately-scheduling descriptor for the
// given process.
func NewImmediateDescriptor(process Process) *ImmediateDescriptor {
	return &ImmediateDescriptor{
		process: process,
	}
}

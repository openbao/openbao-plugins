package scheduler

import (
	"context"
)

// Segment is a bounded executor for processes.
//
// It manages a slice of descriptors, which are responsible for emitting the
// processes to schedule. Each descriptor is run concurrently when the segment
// is started.
//
// When all descriptors terminate, a segment automatically terminates. It
// attempts to complete all remaining work before terminating.
//
// The concurrency of the segment defines the size of the worker pool that
// handle processes. If all workers are busy, the channel used by descriptors to
// emit processes will block until a process completes.
type Segment struct {
	concurrency             int
	descriptors             []Descriptor
	descriptorErrorBehavior ErrorBehavior
	processErrorBehavior    ErrorBehavior
}

var _ Lifecycle = &Segment{}

// WithErrorBehavior sets the error behavior for descriptors and processes for
// this segment.
func (s *Segment) WithErrorBehavior(behavior ErrorBehavior) *Segment {
	return s.WithDescriptorErrorBehavior(behavior).WithProcessErrorBehavior(behavior)
}

// WithDescriptorErrorBehavior sets the error behavior for descriptors in this
// segment.
func (s *Segment) WithDescriptorErrorBehavior(behavior ErrorBehavior) *Segment {
	s.descriptorErrorBehavior = behavior
	return s
}

// WithProcessErrorBehavior sets the error behavior for processes run by this
// segment.
func (s *Segment) WithProcessErrorBehavior(behavior ErrorBehavior) *Segment {
	s.processErrorBehavior = behavior
	return s
}

// Start starts this segment, creating a worker pool of size equal to the
// concurrency of this segment and executing all descriptors.
func (s *Segment) Start(opts LifecycleStartOptions) StartedLifecycle {
	pc := make(chan Process)

	// Bind the lifecycle all the way up here so we can close it in the worker.
	ch := make(chan StartedLifecycle, 1)

	worker := SchedulableFunc(func(ctx context.Context, er ErrorReporter) {
		for {
			select {
			case proc, ok := <-pc:
				if !ok {
					return
				}

				SchedulableProcess(proc).Run(ctx, er)
			case <-ctx.Done():
				// We want to let the lifecycle know that we're preparing to
				// exit, but we want to handle any stragglers left on the
				// process channel, so we'll just close the lifecycle itself
				// rather than closing the channel or blindly exiting.
				if slc, ok := <-ch; ok {
					slc.Close()
					close(ch)
				}
			}
		}
	})

	// This scheduler runs all the descriptors.
	ds := NewScheduler(ManySchedulableDescriptor(s.descriptors, pc)).
		WithErrorBehavior(s.descriptorErrorBehavior).
		WithEventHandler(&SchedulerEventHandlerFuncs{
			OnDoneFunc: func() { close(pc) },
		})

	// This scheduler runs our worker pool.
	ps := NewScheduler(NManySchedulable(s.concurrency, worker)).
		WithErrorBehavior(s.processErrorBehavior)

	// We aggregate them into one scheduler that end users can manage.
	slc := NewParent(ds, ps).WithErrorBehavior(ErrorBehaviorCollect).Start(opts)

	// Let the worker know that we have the lifecycle now.
	ch <- slc

	return slc
}

// NewSegment creates a new segment with the given worker pool size
// (concurrency) and slice of descriptors.
func NewSegment(concurrency int, descriptors []Descriptor) *Segment {
	return &Segment{
		concurrency:             concurrency,
		descriptors:             descriptors,
		descriptorErrorBehavior: ErrorBehaviorTerminate,
		processErrorBehavior:    ErrorBehaviorCollect,
	}
}

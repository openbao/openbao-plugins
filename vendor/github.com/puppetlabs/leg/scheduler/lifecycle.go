package scheduler

import (
	"context"

	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

// StartedLifecycle represents a fully configured, operating scheduler.
type StartedLifecycle interface {
	// Done returns a channel that closes when the lifecycle terminates.
	Done() <-chan struct{}

	// Errs returns the errors associated with this lifecycle. If the lifecycle
	// is not yet closed, this method returns nil.
	Errs() []error

	// Close terminates descriptors, dropping any processes emitted by those
	// descriptors, and asks any running processes to terminate.
	Close()
}

// LifecycleStartOptions are the options all lifecycles must handle when
// starting.
type LifecycleStartOptions struct {
	// Capturer is the error capturer for this lifecycle.
	Capturer trackers.Capturer
}

// Lifecycle represents a partially or fully configured scheduler instance.
// Starting a lifecycle will dispatch the descriptors attached to the given
// lifecycle.
type Lifecycle interface {
	// Start starts all the descriptors for the lifecycle and begins handling
	// the processes for them.
	Start(opts LifecycleStartOptions) StartedLifecycle
}

// SchedulableLifecycle adapts a lifecycle to the Schedulable interface.
func SchedulableLifecycle(l Lifecycle, opts LifecycleStartOptions) Schedulable {
	return SchedulableFunc(func(ctx context.Context, er ErrorReporter) {
		sl := l.Start(opts)

		select {
		case <-ctx.Done():
			sl.Close()
			<-sl.Done()
		case <-sl.Done():
		}

		for _, err := range sl.Errs() {
			er.Put(err)
		}
	})
}

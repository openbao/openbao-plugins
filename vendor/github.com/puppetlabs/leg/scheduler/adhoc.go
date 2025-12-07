package scheduler

import (
	"context"
	"sync"
)

// adhocProcess wraps a process with a channel that can be used for notifying a
// caller of the process result.
type adhocProcess struct {
	ch       chan<- error
	delegate Process
}

func (ap *adhocProcess) Description() string {
	return ap.delegate.Description()
}

func (ap *adhocProcess) Run(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = coercePanic(r)

			// Re-panic after we capture the error.
			defer panic(r)
		}

		ap.ch <- err
	}()

	return ap.delegate.Run(ctx)
}

// AdhocDescriptor is a descriptor that allows external access to submit work to
// be scheduled. It is paired with an AdhocSubmitter, which should be provided
// to external clients to receive the work.
//
// This descriptor is non-blocking; it will indefinitely queue work, consuming a
// proportional amount of memory per pending process if the scheduler does not
// have availability. You may want to rate limit submissions.
type AdhocDescriptor struct {
	queue []*adhocProcess
	cond  *sync.Cond
}

var _ Descriptor = &AdhocDescriptor{}

func (ad *AdhocDescriptor) runOnce(ctx context.Context) (*adhocProcess, bool) {
	ad.cond.L.Lock()
	defer ad.cond.L.Unlock()

	for len(ad.queue) == 0 {
		select {
		case <-ctx.Done():
			return nil, false
		default:
		}

		ad.cond.Wait()
	}

	// Pluck the first item. We zero it out in the queue to make sure we can
	// garbage collect the struct when it's done processing.
	next := ad.queue[0]

	ad.queue[0] = nil
	ad.queue = ad.queue[1:]

	return next, true
}

// Run executes this descriptor with the given process channel.
func (ad *AdhocDescriptor) Run(ctx context.Context, pc chan<- Process) error {
	doneCh := make(chan struct{})
	defer close(doneCh)

	go func() {
		select {
		case <-doneCh:
		case <-ctx.Done():
			// There is a slight inefficiency here because we need to make sure
			// we only wake up the descriptor waiting in the current context,
			// but we don't know which one that is, so we have to broadcast.
			ad.cond.L.Lock()
			defer ad.cond.L.Unlock()

			ad.cond.Broadcast()
		}
	}()

	for {
		p, ok := ad.runOnce(ctx)
		if !ok {
			break
		}

		pc <- p
	}

	return nil
}

// AdhocSubmitter is used to submit work to an adhoc descriptor.
//
// Work is always immediately enqueued.
type AdhocSubmitter struct {
	target *AdhocDescriptor
}

// QueueLen returns the number of work items in the descriptor's queue. These
// items have not yet been submitted to the scheduler for processing.
func (as *AdhocSubmitter) QueueLen() int {
	as.target.cond.L.Lock()
	defer as.target.cond.L.Unlock()

	return len(as.target.queue)
}

// Submit adds a new work item to the descriptor's queue.
func (as *AdhocSubmitter) Submit(p Process) <-chan error {
	as.target.cond.L.Lock()
	defer as.target.cond.L.Unlock()

	ch := make(chan error, 1)

	as.target.queue = append(as.target.queue, &adhocProcess{delegate: p, ch: ch})
	as.target.cond.Signal()

	return ch
}

// NewAdhocDescriptor returns a bound pair of adhoc descriptor and submitter.
// Submitting work items through the returned submitter will enqueue them to the
// returned descriptor.
func NewAdhocDescriptor() (*AdhocDescriptor, *AdhocSubmitter) {
	ad := &AdhocDescriptor{cond: sync.NewCond(&sync.Mutex{})}
	as := &AdhocSubmitter{target: ad}

	return ad, as
}

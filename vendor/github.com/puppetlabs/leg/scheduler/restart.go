package scheduler

import (
	"container/list"
	"context"
	"sync"
)

// RestartableDescriptor provides a mechanism for interrupting a delegate
// descriptor. It will cancel the context of any running delegates that it
// manages. When those delegates then exit, they will be automatically started
// again.
//
// This descriptor does not return until its parent context is done or its
// delegate descriptor exits with an error. If its delegate descriptor finishes
// without an error, it can be restarted.
type RestartableDescriptor struct {
	delegate Descriptor

	mut     sync.RWMutex
	cancels *list.List
}

func (rd *RestartableDescriptor) runOnce(ctx context.Context, pc chan<- Process) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var e *list.Element
	func() {
		rd.mut.Lock()
		defer rd.mut.Unlock()
		e = rd.cancels.PushBack(cancel)
	}()
	defer func() {
		rd.mut.Lock()
		defer rd.mut.Unlock()
		rd.cancels.Remove(e)
	}()

	if err := rd.delegate.Run(ctx, pc); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func (rd *RestartableDescriptor) cancel() {
	rd.mut.RLock()
	defer rd.mut.RUnlock()

	for e := rd.cancels.Front(); e != nil; e = e.Next() {
		e.Value.(context.CancelFunc)()
	}
}

func (rd *RestartableDescriptor) Run(ctx context.Context, pc chan<- Process) error {
	for {
		if err := rd.runOnce(ctx, pc); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
}

// NewRestartableDescriptor returns a new descriptor that can be restarted by
// calling a function.
func NewRestartableDescriptor(delegate Descriptor) (*RestartableDescriptor, func()) {
	rd := &RestartableDescriptor{
		delegate: delegate,
		cancels:  list.New(),
	}

	return rd, rd.cancel
}

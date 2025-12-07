package scheduler

import (
	"sync"
)

var (
	// ErrorBehaviorCollect allows all processes to complete and returns a full
	// set of errors when they have finished.
	ErrorBehaviorCollect ErrorBehavior = &errorBehaviorCollect{}

	// ErrorBehaviorTerminate causes the entire lifecycle to clean up and exit
	// when the first error occurs.
	ErrorBehaviorTerminate ErrorBehavior = &errorBehaviorTerminate{}

	// ErrorBehaviorDrop ignores errors, merely logging them for reference.
	ErrorBehaviorDrop ErrorBehavior = &errorBehaviorDrop{}
)

// ErrorReporter allows reporter processes to submit errors to a handler created
// by a particular error behavior.
type ErrorReporter interface {
	Put(err error)
}

// ErrorHandler defines a contract that a Scheduler uses to manage errors. When
// the error handler's Done channel closes, the scheduler begins the process of
// terminating. The errors returned by the error handler then become the errors
// returned by the scheduler.
type ErrorHandler interface {
	ErrorReporter

	// Done returns a channel that closes when this error handler cannot accept
	// further submissions.
	Done() <-chan struct{}

	// Errors returns a copy of the errors collected by this error handler.
	Errs() []error
}

// Most of these handlers never close their channel, so we just keep a single
// open channel hanging around.
var ch = make(chan struct{})

// ErrorBehavior defines the way a lifecycle handles errors from descriptors and
// processes.
type ErrorBehavior interface {
	// NewHandler returns a new handler that provides the desired error
	// management functionality.
	NewHandler() ErrorHandler
}

type errorHandlerCollect struct {
	errs []error
	mut  sync.RWMutex
}

func (h *errorHandlerCollect) Put(err error) {
	h.mut.Lock()
	defer h.mut.Unlock()

	h.errs = append(h.errs, err)
}

func (h *errorHandlerCollect) Done() <-chan struct{} {
	// This handler never terminates.
	return ch
}

func (h *errorHandlerCollect) Errs() []error {
	h.mut.RLock()
	defer h.mut.RUnlock()

	return append([]error{}, h.errs...)
}

type errorBehaviorCollect struct{}

func (errorBehaviorCollect) NewHandler() ErrorHandler {
	return &errorHandlerCollect{}
}

type errorHandlerTerminate struct {
	ch  chan struct{}
	err error
	mut sync.RWMutex
}

func (h *errorHandlerTerminate) Put(err error) {
	h.mut.Lock()
	defer h.mut.Unlock()

	if h.err != nil {
		return
	}

	close(h.ch)
	h.err = err
}

func (h *errorHandlerTerminate) Done() <-chan struct{} {
	return h.ch
}

func (h *errorHandlerTerminate) Errs() []error {
	h.mut.RLock()
	defer h.mut.RUnlock()

	if h.err == nil {
		return nil
	}

	return []error{h.err}
}

type errorBehaviorTerminate struct{}

func (errorBehaviorTerminate) NewHandler() ErrorHandler {
	return &errorHandlerTerminate{
		ch: make(chan struct{}),
	}
}

type errorHandlerDrop struct{}

func (h *errorHandlerDrop) Put(err error)         {}
func (h *errorHandlerDrop) Done() <-chan struct{} { return ch }
func (h *errorHandlerDrop) Errs() []error         { return nil }

type errorBehaviorDrop struct{}

func (errorBehaviorDrop) NewHandler() ErrorHandler {
	return &errorHandlerDrop{}
}

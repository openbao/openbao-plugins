package scheduler

import (
	"context"

	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

// SchedulerEventHandler defines an interface by which a client can respond to
// lifecycle events.
type SchedulerEventHandler interface {
	// OnDone is fired when the scheduler terminates.
	OnDone()
}

// SchedulerEventHandlerFuncs allows a client to partially implement an event
// handler, choosing to receiving only the events they provide handlers for.
type SchedulerEventHandlerFuncs struct {
	// OnDoneFunc is the function to call when the scheduler terminates.
	OnDoneFunc func()
}

// OnDone fires OnDoneFunc if non-nil.
func (sehf *SchedulerEventHandlerFuncs) OnDone() {
	if sehf.OnDoneFunc == nil {
		return
	}

	sehf.OnDoneFunc()
}

type startedScheduler struct {
	ctx           context.Context
	cancel        context.CancelFunc
	children      []chan struct{}
	waiter        chan struct{}
	errorHandler  ErrorHandler
	eventHandlers []SchedulerEventHandler
}

func (ss *startedScheduler) Done() <-chan struct{} {
	return ss.waiter
}

func (ss *startedScheduler) Errs() []error {
	select {
	case <-ss.waiter:
		return ss.errorHandler.Errs()
	default:
		return nil
	}
}

func (ss *startedScheduler) Close() {
	ss.cancel()
}

func (ss *startedScheduler) supervise(i int, fn func(ctx context.Context, i int, er ErrorReporter)) {
	defer func() {
		close(ss.children[i])
	}()

	fn(ss.ctx, i, ss.errorHandler)
}

func (ss *startedScheduler) wait() {
	func() {
		defer close(ss.waiter)

		for _, c := range ss.children {
			select {
			case <-c:
			case <-ss.errorHandler.Done():
				ss.cancel()
				<-c
			}
		}
	}()

	for _, eh := range ss.eventHandlers {
		eh.OnDone()
	}
}

// Scheduler provides a generic mechanism for managing parallel work. It extends
// Goroutines by providing consistent error handling and external termination.
//
// All lifecycles in this package use one or more schedulers to execute.
// Generally, you should not need to use this lifecycle, but should prefer one
// of the other more concrete lifecycles. However, in conjuction with the
// Schedulable interface adapters for lifecycles, descriptors, and processes, it
// is relatively easy to implement new scheduling algorithms if desired.
type Scheduler struct {
	children      ManySchedulable
	errorBehavior ErrorBehavior
	eventHandlers []SchedulerEventHandler
}

var _ Lifecycle = &Scheduler{}

// WithErrorBehavior sets the error behavior for this scheduler.
func (s *Scheduler) WithErrorBehavior(behavior ErrorBehavior) *Scheduler {
	s.errorBehavior = behavior
	return s
}

// WithEventHandler adds an event handler to the list of event handlers to
// trigger.
func (s *Scheduler) WithEventHandler(handler SchedulerEventHandler) *Scheduler {
	s.eventHandlers = append(s.eventHandlers, handler)
	return s
}

// Start starts this scheduler, asynchronously dispatching and managing all of
// the provided children.
func (s *Scheduler) Start(opts LifecycleStartOptions) StartedLifecycle {
	ctx, cancel := context.WithCancel(context.Background())

	if opts.Capturer != nil {
		ctx = trackers.NewContextWithCapturer(ctx, opts.Capturer)
	}

	childrenLen := s.children.Len()

	ss := &startedScheduler{
		ctx:           ctx,
		cancel:        cancel,
		children:      make([]chan struct{}, childrenLen),
		waiter:        make(chan struct{}),
		errorHandler:  s.errorBehavior.NewHandler(),
		eventHandlers: s.eventHandlers,
	}

	for i := 0; i < childrenLen; i++ {
		ss.children[i] = make(chan struct{})
		go ss.supervise(i, s.children.Run)
	}

	go ss.wait()

	return ss
}

// NewScheduler creates a scheduler to manage all of the schedulable children
// provided.
func NewScheduler(children ManySchedulable) *Scheduler {
	return &Scheduler{
		children:      children,
		errorBehavior: ErrorBehaviorCollect,
	}
}

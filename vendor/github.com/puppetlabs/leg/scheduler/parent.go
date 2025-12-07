package scheduler

// Parent is a lifecycle that aggregates other lifecycles.
type Parent struct {
	delegates     []Lifecycle
	errorBehavior ErrorBehavior
}

var _ Lifecycle = &Parent{}

// WithErrorBehavior changes the error behavior for the parent. It does not
// affect the error behavior of any delegate lifecycles.
func (p *Parent) WithErrorBehavior(errorBehavior ErrorBehavior) *Parent {
	p.errorBehavior = errorBehavior
	return p
}

// Start starts all the delegate lifecycles that are part of this parent in
// parallel and waits for them to terminate according to the specified error
// behavior.
func (p *Parent) Start(opts LifecycleStartOptions) StartedLifecycle {
	sd := make(ManySchedulableSlice, len(p.delegates))
	for i, d := range p.delegates {
		sd[i] = SchedulableLifecycle(d, opts)
	}

	return NewScheduler(sd).WithErrorBehavior(p.errorBehavior).Start(opts)
}

// NewParent creates a new parent lifecycle comprised of the given delegate
// lifecycles.
func NewParent(delegates ...Lifecycle) *Parent {
	return &Parent{
		delegates:     delegates,
		errorBehavior: ErrorBehaviorTerminate,
	}
}

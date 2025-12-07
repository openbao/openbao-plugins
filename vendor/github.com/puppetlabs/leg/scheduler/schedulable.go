package scheduler

import "context"

// Schedulable represents an individual item of work that can be scheduled in
// this package. It is the most general form of work for managed execution.
type Schedulable interface {
	// Run executes work with the expectation that the work will take all
	// possible measures to cleanly terminate when the given context is done.
	// Any errors that occur may be reported to the given error reporter.
	Run(ctx context.Context, er ErrorReporter)
}

// SchedulableFunc makes the given function conform to the Schedulable
// interface.
type SchedulableFunc func(ctx context.Context, er ErrorReporter)

// Run calls the underlying function.
func (sf SchedulableFunc) Run(ctx context.Context, er ErrorReporter) {
	sf(ctx, er)
}

// ManySchedulable represents multiple items of work, each of which needs to be
// scheduled.
type ManySchedulable interface {
	// Len returns the number of items of work to schedule. The contract of this
	// interface requires that this be a pure function.
	Len() int

	// Run executes the *i*th schedulable work.
	Run(ctx context.Context, i int, er ErrorReporter)
}

// ManySchedulableSlice is an implementation of ManySchedulable for a slice of
// Schedulables.
type ManySchedulableSlice []Schedulable

var _ ManySchedulable = ManySchedulableSlice{}

// Len returns the length of the underlying slice.
func (mss ManySchedulableSlice) Len() int {
	return len(mss)
}

// Run executes the schedulable work at the *i*th element in the underlying
// slice.
func (mss ManySchedulableSlice) Run(ctx context.Context, i int, er ErrorReporter) {
	mss[i].Run(ctx, er)
}

type manySchedulablePool struct {
	size     int
	delegate Schedulable
}

func (msp manySchedulablePool) Len() int {
	return msp.size
}

func (msp manySchedulablePool) Run(ctx context.Context, i int, er ErrorReporter) {
	msp.delegate.Run(ctx, er)
}

// NManySchedulable creates a new worker pool of the given size. The pool will
// execute the given delegate exactly the number of times as requested by the
// provided size.
func NManySchedulable(size int, delegate Schedulable) ManySchedulable {
	return &manySchedulablePool{
		size:     size,
		delegate: delegate,
	}
}

// OneSchedulable conforms a single schedulable work item to the ManySchedulable
// interface.
func OneSchedulable(s Schedulable) ManySchedulable {
	return NManySchedulable(1, s)
}

package retry

import (
	"context"
	"time"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/timeutil/pkg/backoff"
	"github.com/puppetlabs/leg/timeutil/pkg/clock"
	"github.com/puppetlabs/leg/timeutil/pkg/clockctx"
)

// DefaultBackoffFactory is a networking-appropriate backoff that the wait
// functions use by default. It provides a non-sliding exponential backoff
// starting at 250ms, factor 2, a 30 second cap, and full jitter using the
// default RNG.
var DefaultBackoffFactory = backoff.Build(
	backoff.Exponential(250*time.Millisecond, 2.0),
	backoff.MaxBound(30*time.Second),
	backoff.FullJitter(),
	backoff.NonSliding,
)

// WaitOptions allows the behavior of Wait to be customized.
type WaitOptions struct {
	// BackoffFactory is the backoff algorithm to use for waiting. If not
	// specified, a sensible default appropriate for network communication will
	// be chosen: a non-sliding exponential backoff with factor 2 and full
	// jitter.
	BackoffFactory *backoff.Factory

	// Clock is the clock implementation used to perform the backoff.
	Clock clock.Clock
}

// WaitOption is a setter for one or more wait options.
type WaitOption interface {
	// ApplyToWaitOptions configures the specified wait options for this option.
	ApplyToWaitOptions(target *WaitOptions)
}

// ApplyOptions runs each of the given options against this options struct.
func (o *WaitOptions) ApplyOptions(opts []WaitOption) {
	for _, opt := range opts {
		opt.ApplyToWaitOptions(o)
	}
}

// WaitOptionFunc allows a function to be used as a wait option.
type WaitOptionFunc func(target *WaitOptions)

var _ WaitOption = WaitOptionFunc(nil)

// ApplyToWaitOptions configures the specified wait options by calling this
// function.
func (wof WaitOptionFunc) ApplyToWaitOptions(target *WaitOptions) {
	wof(target)
}

// WithBackoffFactory changes the backoff algorithm to the specified one.
func WithBackoffFactory(bf *backoff.Factory) WaitOption {
	return WaitOptionFunc(func(target *WaitOptions) {
		target.BackoffFactory = bf
	})
}

// WithClock changes the backoff clock to the specified one.
func WithClock(c clock.Clock) WaitOption {
	return WaitOptionFunc(func(target *WaitOptions) {
		target.Clock = c
	})
}

// WorkFunc is a function that performs an arbitrary operation. If the operation
// needs to be retried for any reason, the function must return false.
type WorkFunc func(ctx context.Context) (bool, error)

// Repeat indicates that work should be retried.
func Repeat(err error) (bool, error) {
	return false, err
}

// Done indicates that work should not be retried.
func Done(err error) (bool, error) {
	return true, err
}

// RepeatUnlessMarkedUser indicates that work should be retried if an error is
// provided and it is not marked as a user error.
func RepeatUnlessMarkedUser(err error) (bool, error) {
	if err == nil || errmark.MarkedUser(err) {
		return true, err
	}

	return false, err
}

// DoneUnlessMarkedTransient indicates that work should not be retried unless
// the given error is marked transient.
func DoneUnlessMarkedTransient(err error) (bool, error) {
	if errmark.MarkedTransient(err) {
		return false, err
	}

	return true, err
}

// Wait runs a given work function under a context with a particular backoff
// algorithm if the work needs to be retried.
//
// Each time the work is attempted, this function sets its return value to the
// error produced by the work. If the context expires and the work has not
// returned an error, the context error is returned instead.
func Wait(ctx context.Context, work WorkFunc, opts ...WaitOption) (err error) {
	o := &WaitOptions{
		BackoffFactory: DefaultBackoffFactory,
		Clock:          clockctx.Clock(ctx),
	}
	o.ApplyOptions(opts)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = clockctx.WithClock(ctx, o.Clock)

	b, err := o.BackoffFactory.New()
	if err != nil {
		return
	}

	bv, err := b.Next(ctx)
	if err != nil {
		return
	}

	t := o.Clock.NewTimer(bv)
	defer func() { t.Stop() }()

	for {
		select {
		case <-t.C():
		case <-ctx.Done():
			if err == nil {
				err = ctx.Err()
			}
			return
		}

		// It's possible that t.C() and ctx.Done() happen around the same time,
		// so we'll check the context one more time.
		select {
		case <-ctx.Done():
			if err == nil {
				err = ctx.Err()
			}
			return
		default:
		}

		var ok bool
		ok, err = work(ctx)
		if ok {
			return
		}

		bv, berr := b.Next(ctx)
		if berr != nil {
			if err == nil {
				err = berr
			}
			return
		}

		t.Reset(bv)
	}
}

// WaitAsync runs a given work function in a separate Goroutine, but otherwise
// behaves identically to Wait.
func WaitAsync(ctx context.Context, work WorkFunc, opts ...WaitOption) <-chan error {
	ch := make(chan error, 1)
	go func() {
		ch <- Wait(ctx, work, opts...)
	}()
	return ch
}

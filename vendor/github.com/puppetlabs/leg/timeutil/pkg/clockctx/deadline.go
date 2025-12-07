// Portions of this file are derived from Go's context package.
//
// https://go.googlesource.com/go/+/go1.14.15/src/context/context.go
//
// Licensed under a three-clause BSD-style license. A copy of the full license
// document is included in this distribution in the file `deadline.go.LICENSE`.

package clockctx

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/puppetlabs/leg/timeutil/pkg/clock"
)

// closedChan is a reusable closed channel.
var closedChan = make(chan struct{})

func init() {
	close(closedChan)
}

// WithDeadlineOfClock sets the given context deadline to be no later than the
// given time under the auspices of a particular clock.
//
// This function delegates to context.WithDeadline if c == clock.RealClock.
// Otherwise, it is vastly less efficient than the standard library's context
// package equivalent because it does not have access to internal data
// associated with those contexts. In performance-sensitive applications, this
// function must be used with care.
//
// This function does not set the clock in the context. To do that, use
// clockctx.WithDeadline(clockctx.WithClock(ctx, c), d) instead.
//
// Note that using this method in combination with a clock that moves slower
// than wall time and downstream calls to the standard library
// context.WithDeadline/context.WithTimeout methods may result in strange,
// undesirable behavior.
func WithDeadlineOfClock(ctx context.Context, c clock.Clock, d time.Time) (context.Context, context.CancelFunc) {
	if c == clock.RealClock {
		return context.WithDeadline(ctx, d)
	}
	// Note that this implementation does not shortcut if the deadline of the
	// parent is before the requested time, as the actual passage of time among
	// implementations of clock.Clock may vary. However, it will reduce the
	// deadline to the parent's.
	if cur, ok := ctx.Deadline(); ok && cur.Before(d) {
		d = cur
	}
	tctx := &timerCtx{
		Context:  ctx,
		clock:    c,
		deadline: d,
	}
	propagateCancel(ctx, tctx)
	dur := d.Sub(c.Now())
	if dur <= 0 {
		tctx.cancel(context.DeadlineExceeded) // deadline has already passed
		return tctx, func() { tctx.cancel(context.Canceled) }
	}
	tctx.mu.Lock()
	defer tctx.mu.Unlock()
	if tctx.err == nil {
		tctx.timer = clock.AfterFunc(c, dur, func() {
			tctx.cancel(context.DeadlineExceeded)
		})
	}
	return tctx, func() { tctx.cancel(context.Canceled) }
}

// WithTimeoutOfClock is equivalent to WithDeadlineOfClock(ctx, c,
// c.Now().Add(timeout)).
func WithTimeoutOfClock(ctx context.Context, c clock.Clock, timeout time.Duration) (context.Context, context.CancelFunc) {
	return WithDeadlineOfClock(ctx, c, c.Now().Add(timeout))
}

// WithDeadline behaves exactly like context.WithDeadline but uses the clock
// from the context, if any, to determine which clock to use for the deadline.
//
// This function delegates to WithDeadlineOfClock and has all of the caveats
// associated with it.
func WithDeadline(ctx context.Context, d time.Time) (context.Context, context.CancelFunc) {
	return WithDeadlineOfClock(ctx, Clock(ctx), d)
}

// WithTimeout behaves exactly like context.WithTimeout but uses the clock from
// the context, if any, to determine which clock to use for the timeout.
//
// This function delegates to WithTimeoutOfClock and has all of the caveats
// associated with it.
func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return WithTimeoutOfClock(ctx, Clock(ctx), timeout)
}

// propagateCancel arranges for child to be canceled when parent is.
func propagateCancel(parent context.Context, child *timerCtx) {
	done := parent.Done()
	if done == nil {
		return // parent is never canceled
	}

	select {
	case <-done:
		// parent is already canceled
		child.cancel(parent.Err())
		return
	default:
	}

	go func() {
		select {
		case <-parent.Done():
			child.cancel(parent.Err())
		case <-child.Done():
		}
	}()
}

// A timerCtx carries a timer and a deadline. It embeds the implementation of
// the Go library's cancelCtx to implement Done and Err as well.
type timerCtx struct {
	context.Context
	mu    sync.Mutex    // protects following fields
	done  chan struct{} // created lazily, closed by first cancel call
	err   error         // set to non-nil by the first cancel call
	clock clock.Clock
	timer clock.Timer

	deadline time.Time
}

func (c *timerCtx) Deadline() (deadline time.Time, ok bool) {
	return c.deadline, true
}

func (c *timerCtx) Done() <-chan struct{} {
	c.mu.Lock()
	if c.done == nil {
		c.done = make(chan struct{})
	}
	d := c.done
	c.mu.Unlock()
	return d
}

func (c *timerCtx) Err() error {
	c.mu.Lock()
	err := c.err
	c.mu.Unlock()
	return err
}

func (c *timerCtx) String() string {
	return contextName(c.Context) + ".WithDeadline(" +
		c.deadline.String() + " [" +
		c.deadline.Sub(c.clock.Now()).String() + "])"
}

func (c *timerCtx) cancel(err error) {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return // already canceled
	}
	c.err = err
	if c.done == nil {
		c.done = closedChan
	} else {
		close(c.done)
	}
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.mu.Unlock()
}

func contextName(c context.Context) string {
	if s, ok := c.(fmt.Stringer); ok {
		return s.String()
	}
	return reflect.TypeOf(c).String()
}

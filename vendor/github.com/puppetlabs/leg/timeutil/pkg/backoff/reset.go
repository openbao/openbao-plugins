package backoff

import (
	"context"
	"time"

	"github.com/puppetlabs/leg/timeutil/pkg/clock"
	"github.com/puppetlabs/leg/timeutil/pkg/clockctx"
)

type resetAfterGenerator struct {
	cur      *Backoff
	called   time.Time
	delegate *Factory
	d        time.Duration
	clock    clock.PassiveClock
}

func (rag *resetAfterGenerator) Next(ctx context.Context) (d time.Duration, err error) {
	c := rag.clock
	if c == nil {
		c = clockctx.Clock(ctx)
	}

	called := c.Now()
	defer func() {
		rag.called = called
	}()

	if rag.cur == nil || c.Since(rag.called) >= rag.d {
		rag.cur, err = rag.delegate.New()
		if err != nil {
			return
		}
	}

	return rag.cur.Next(ctx)
}

type resetAfterGeneratorFactory struct {
	delegate *Factory
	d        time.Duration
	clock    clock.PassiveClock
}

func (ragf *resetAfterGeneratorFactory) New() (Generator, error) {
	return &resetAfterGenerator{
		delegate: ragf.delegate,
		d:        ragf.d,
		clock:    ragf.clock,
	}, nil
}

// ResetAfterOptions allows the behavior of ResetAfter to be customized.
type ResetAfterOptions struct {
	// Clock is the clock implementation used to check the elapsed duration.
	Clock clock.PassiveClock
}

// ResetAfterOption is a setter for one or more reset-after algorithm options.
type ResetAfterOption interface {
	// ApplyToResetAfterOptions configures the specified resetafter options for this option.
	ApplyToResetAfterOptions(target *ResetAfterOptions)
}

// ApplyOptions runs each of the given options against this options struct.
func (o *ResetAfterOptions) ApplyOptions(opts []ResetAfterOption) {
	for _, opt := range opts {
		opt.ApplyToResetAfterOptions(o)
	}
}

// ResetAfterOptionFunc allows a function to be used as a reset-after algorithm
// option.
type ResetAfterOptionFunc func(target *ResetAfterOptions)

var _ ResetAfterOption = ResetAfterOptionFunc(nil)

// ApplyToResetAfterOptions configures the specified reset-after algorithm
// options by calling this function.
func (raof ResetAfterOptionFunc) ApplyToResetAfterOptions(target *ResetAfterOptions) {
	raof(target)
}

// ResetAfterWithClock changes the clock to the specified one.
func ResetAfterWithClock(c clock.PassiveClock) ResetAfterOption {
	return ResetAfterOptionFunc(func(target *ResetAfterOptions) {
		target.Clock = c
	})
}

// ResetAfter creates a generator factory that replaces a delegate backoff
// factory with a fresh instance (i.e., resets its internal state) after a
// certain amount of time has elapsed between calls to Next().
func ResetAfter(delegate *Factory, d time.Duration, opts ...ResetAfterOption) GeneratorFactory {
	o := &ResetAfterOptions{}
	o.ApplyOptions(opts)

	return &resetAfterGeneratorFactory{
		delegate: delegate,
		d:        d,
		clock:    o.Clock,
	}
}

package backoff

import (
	"context"
	"time"
)

// Backoff is a combination of a generator of backoff durations and zero or more
// rules that adjust the generated duration amounts.
//
// Methods of this struct are not safe to use across Goroutines, although they
// may safely guard work done in other Goroutines.
type Backoff struct {
	g Generator
	r Rule
}

// Next provides a new backoff amount to delay work by.
func (b *Backoff) Next(ctx context.Context) (next time.Duration, err error) {
	generate := true
	if b.r != nil {
		generate, next, err = b.r.ApplyBefore()
		if err != nil {
			return 0, err
		}
	}

	if generate {
		next, err = b.g.Next(ctx)
		if err != nil {
			return 0, err
		}
	}

	if b.r != nil {
		next, err = b.r.ApplyAfter(next)
		if err != nil {
			return 0, err
		}
	}

	return
}

// Once creates a one-time-use backoff using the given generator factory and
// rule factories.
func Once(gf GeneratorFactory, rfs ...RuleFactory) (*Backoff, error) {
	return Build(gf, rfs...).New()
}

// Factory is a Goroutine-safe factory for creating backoff instances.
type Factory struct {
	gf  GeneratorFactory
	rfs []RuleFactory
}

// New creates a new backoff instance.
func (af *Factory) New() (*Backoff, error) {
	g, err := af.gf.New()
	if err != nil {
		return nil, err
	}

	rs := make(Rules, len(af.rfs))
	for i, rf := range af.rfs {
		r, err := rf.New()
		if err != nil {
			return nil, err
		}

		rs[i] = r
	}

	b := &Backoff{
		g: g,
	}

	if injector, ok := b.g.(RuleInjector); ok {
		injector.InjectRule(rs)
	} else {
		b.r = rs
	}

	return b, nil
}

// Build creates a Goroutine-safe factory for backoffs from a generator factory
// and zero or more rule factories.
func Build(gf GeneratorFactory, rfs ...RuleFactory) *Factory {
	return &Factory{
		gf:  gf,
		rfs: rfs,
	}
}

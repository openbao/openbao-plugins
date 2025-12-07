package clockctx

import (
	"context"

	"github.com/puppetlabs/leg/timeutil/pkg/clock"
)

type contextKey int

const (
	clockContextKey contextKey = iota
)

// WithClock sets the given clock in the context.
func WithClock(ctx context.Context, c clock.Clock) context.Context {
	return context.WithValue(ctx, clockContextKey, c)
}

// Clock retrieves the current clock set in the context, defaulting to
// clock.RealClock.
func Clock(ctx context.Context) clock.Clock {
	c, ok := ctx.Value(clockContextKey).(clock.Clock)
	if !ok {
		c = clock.RealClock
	}

	return c
}

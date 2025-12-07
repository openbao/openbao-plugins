package trackers

import (
	"context"
	"fmt"
)

type contextKey int

const (
	capturerContextKey contextKey = iota
)

func NewContextWithCapturer(ctx context.Context, c Capturer) context.Context {
	return context.WithValue(ctx, capturerContextKey, c)
}

func CapturerFromContext(ctx context.Context) (c Capturer, ok bool) {
	c, ok = ctx.Value(capturerContextKey).(Capturer)
	return
}

func MustCapturerFromContext(ctx context.Context) Capturer {
	c, ok := CapturerFromContext(ctx)
	if !ok {
		panic(fmt.Errorf("no capturer in context"))
	}

	return c
}

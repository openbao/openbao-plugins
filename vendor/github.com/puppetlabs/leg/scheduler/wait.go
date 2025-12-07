package scheduler

import (
	"context"
)

// WaitContext waits until the given lifecycle completes or the context is done.
// If the context is canceled first, the context error is returned.
func WaitContext(ctx context.Context, lc StartedLifecycle) error {
	select {
	case <-lc.Done():
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// CloseWaitContext terminates the given lifecycle and then gracefully waits for
// it to shut down.
func CloseWaitContext(ctx context.Context, lc StartedLifecycle) error {
	lc.Close()
	return WaitContext(ctx, lc)
}

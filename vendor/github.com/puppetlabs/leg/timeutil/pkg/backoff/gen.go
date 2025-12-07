package backoff

import (
	"context"
	"time"
)

// Generator produces new durations based on a particular algorithm.
type Generator interface {
	// Next returns the next duration to back off by.
	Next(ctx context.Context) (time.Duration, error)
}

// GeneratorFactory provides a Goroutine-safe factory for creating generators of
// a particular algorithm.
type GeneratorFactory interface {
	// New creates a generator with this factory's configuration.
	New() (Generator, error)
}

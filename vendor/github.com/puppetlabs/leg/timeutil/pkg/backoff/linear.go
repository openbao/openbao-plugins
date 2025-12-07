package backoff

import (
	"context"
	"math"
	"time"
)

type linearGenerator struct {
	n    int64
	step time.Duration
}

func (lg *linearGenerator) Next(ctx context.Context) (time.Duration, error) {
	if lg.n < math.MaxInt64 {
		lg.n++
	}

	return time.Duration(mul64Checked(int64(lg.step), lg.n)), nil
}

type linearGeneratorFactory struct {
	step time.Duration
}

func (lgf *linearGeneratorFactory) New() (Generator, error) {
	return &linearGenerator{
		step: lgf.step,
	}, nil
}

// Linear creates a generator factory that uses the specified step value to use
// to increase the backoff amount each time.
func Linear(step time.Duration) GeneratorFactory {
	return &linearGeneratorFactory{
		step: step,
	}
}

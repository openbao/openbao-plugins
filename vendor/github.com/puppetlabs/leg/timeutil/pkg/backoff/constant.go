package backoff

import (
	"context"
	"time"
)

type constantGenerator struct {
	step time.Duration
}

func (cg *constantGenerator) Next(ctx context.Context) (time.Duration, error) {
	return cg.step, nil
}

type constantGeneratorFactory struct {
	g *constantGenerator
}

func (cgf *constantGeneratorFactory) New() (Generator, error) {
	return cgf.g, nil
}

// Constant creates a generator factory that uses the same given step duration
// for each successive backoff.
func Constant(step time.Duration) GeneratorFactory {
	return &constantGeneratorFactory{
		g: &constantGenerator{
			step: step,
		},
	}
}

// Immediate is a generator factory for no backoff. It is equivalent to
// Constant(0).
var Immediate = Constant(0)

package rand

import (
	exprand "golang.org/x/exp/rand"
)

// ExpRandFactory is a Goroutine-safe factory for random number generators that
// use a source compatible with Go's new experimental rand
// (golang.org/x/exp/rand) package.
type ExpRandFactory struct {
	cons   func(seed uint64) exprand.Source
	seeder Seeder
}

var _ Factory = &ExpRandFactory{}

// New returns a new RNG for this configuration.
func (erf *ExpRandFactory) New() (Rand, error) {
	var seed Seed
	if err := erf.seeder.Seed(&seed); err != nil {
		return nil, err
	}

	rng := exprand.New(erf.cons(uint64(seed)))

	// Check to make sure this conforms to our interface.
	var _ DiscreteRand = rng

	return rng, nil
}

// NewExpRandFactory creates a factory for producing RNGs from a source
// compatible with Go's new experimental rand package initialized with the given
// seeder.
func NewExpRandFactory(cons func(seed uint64) exprand.Source, seeder Seeder) *ExpRandFactory {
	return &ExpRandFactory{
		cons:   cons,
		seeder: seeder,
	}
}

// NewPCGFactory creates a factory for producing RNGs using the permuted
// congruential generator (PCG) algorithm. Each RNG it produces will be seeded
// using the given seeder.
func NewPCGFactory(seeder Seeder) *ExpRandFactory {
	return NewExpRandFactory(exprand.NewSource, seeder)
}

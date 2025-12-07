// Package rand provides safer defaults and more options than the built-in Go
// random packages.
package rand

import "io"

// Rand defines an instance of a random number generator. Instances of random
// number generators are generally not safe to use across Goroutines. Exceptions
// are indicated as such.
type Rand interface {
	io.Reader
}

// DiscreteRand is a random number generator that natively supports generating
// integers in addition to producing random byte data.
type DiscreteRand interface {
	Rand

	// Uint64 returns the next available random integer.
	Uint64() uint64
}

// Factory is a Goroutine-safe factory for constructing and seeding random
// number generators using a particular algorithm.
type Factory interface {
	// New creates an RNG.
	New() (Rand, error)
}

// FactoryFunc allows a factory to be defined by a function.
type FactoryFunc func() (Rand, error)

var _ Factory = FactoryFunc(nil)

// New creates an RNG by calling the function.
func (ff FactoryFunc) New() (Rand, error) {
	return ff()
}

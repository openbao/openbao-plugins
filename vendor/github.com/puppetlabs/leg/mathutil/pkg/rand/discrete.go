// Portions of this file are derived from the Go experimental random library.
//
// https://github.com/golang/exp/blob/e89b829e73ea9e1c72dda449af03683e4a9f8f17/rand/rand.go
//
// Licensed under a three-clause BSD-style license. A copy of the full license
// document is included in this distribution in the file `discrete.go.LICENSE`.

package rand

import (
	"encoding/binary"
	"math"
)

// Uint64 returns a pseudo-random 64-bit integer as a uint64.
func Uint64(rng Rand) (r uint64, err error) {
	if drng, ok := rng.(DiscreteRand); ok {
		return drng.Uint64(), nil
	}

	err = binary.Read(rng, binary.BigEndian, &r)
	return
}

// Uint64N returns, as a uint64, a pseudo-random number in [0,n).
func Uint64N(rng Rand, n uint64) (uint64, error) {
	if n == 0 {
		return 0, ErrImpossible
	}

	r, err := Uint64(rng)
	if err != nil {
		return 0, err
	}

	if n&(n-1) == 0 {
		r &= n - 1
		return r, nil
	}

	// If n does not divide r, to avoid modulo bias we must not use r that is
	// within math.MaxUint64%n of the top of the range.
	if r > math.MaxUint64-n { // Fast check.
		ceiling := math.MaxUint64 - math.MaxUint64%n
		for r >= ceiling {
			r, err = Uint64(rng)
			if err != nil {
				return 0, err
			}
		}
	}

	r %= n
	return r, nil
}

// Uint64Between returns, as a uint64, a pseudo-random number in [lo,hi).
func Uint64Between(rng Rand, lo, hi uint64) (uint64, error) {
	if lo >= hi {
		return 0, ErrImpossible
	}

	rv, err := Uint64N(rng, hi-lo)
	if err != nil {
		return 0, err
	}

	return lo + rv, nil
}

// Float64 returns, as a float64, a pseudo-random number in [0.0,1.0).
func Float64(rng Rand) (r float64, err error) {
	for {
		var iv uint64
		iv, err = Uint64N(rng, 1<<53)
		if err != nil {
			return
		}

		r = float64(iv) / (1 << 53)
		if r == 1.0 {
			// There is one bug in the value stream: r.Int63() may be so close
			// to 1<<63 that the division rounds up to 1.0, and we've guaranteed
			// that the result is always less than 1.0.
			//
			// We tried to fix this by mapping 1.0 back to 0.0, but since
			// float64 values near 0 are much denser than near 1, mapping 1 to 0
			// caused a theoretically significant overshoot in the probability
			// of returning 0. Instead of that, if we round up to 1, just try
			// again. Getting 1 only happens 1/2⁵³ of the time, so most clients
			// will not observe it anyway.
			continue
		}
		return
	}
}

// Float64Between returns, as a float64, a pseudo-random number between [lo,hi).
func Float64Between(rng Rand, lo, hi float64) (float64, error) {
	if lo >= hi {
		return 0, ErrImpossible
	}

	rv, err := Float64(rng)
	if err != nil {
		return 0, err
	}

	return lo + rv*(hi-lo), nil
}

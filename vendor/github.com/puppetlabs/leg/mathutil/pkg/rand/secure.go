package rand

import (
	cryptorand "crypto/rand"
)

type secureRand struct{}

var _ LockableRand = &secureRand{}

func (*secureRand) Read(buf []byte) (int, error) {
	return cryptorand.Read(buf)
}

func (sr *secureRand) ThreadSafe() Rand {
	return sr
}

var (
	// SecureRand is the system cryptographically-secure random number
	// generator. It is equivalent to the Reader of Go's crypto/rand. SecureRand
	// is safe to use across Goroutines.
	SecureRand Rand = &secureRand{}

	// SecureFactory provides SecureRand as a factory for compatibility with
	// users that expect a factory.
	SecureFactory Factory = FactoryFunc(func() (Rand, error) {
		return SecureRand, nil
	})
)

type secureSeeder struct{}

func (ss *secureSeeder) Seed(s Seedable) error {
	seed, err := Uint64(SecureRand)
	if err != nil {
		return err
	}

	s.Seed(seed)
	return nil
}

// SecureSeeder seeds non-secure RNGs with a value produced by SecureRand.
var SecureSeeder Seeder = &secureSeeder{}

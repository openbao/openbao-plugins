package rand

// Seedable is the type of an RNG that is able to be seeded.
type Seedable interface {
	// Seed sets this RNG's seed.
	Seed(seed uint64)
}

// Seeder seeds an RNG with a particular value or algorithm.
//
// All implementations of Seeder must be safe for use across concurrent
// Goroutines.
type Seeder interface {
	// Seed sets the target's seed using this seeder's algorithm.
	Seed(target Seedable) error
}

// Seed is a Seedable that simply stores the seed value to be retrieved later.
type Seed uint64

var _ Seedable = (*Seed)(nil)

// Seed stores the given seed as this type's value.
func (s *Seed) Seed(seed uint64) {
	*s = Seed(seed)
}

package rand

type testRand struct {
	cur uint64
}

var _ DiscreteRand = &testRand{}

func (tr *testRand) Read(buf []byte) (n int, err error) {
	var val uint64
	var rem uint8
	for n = 0; n < len(buf); n++ {
		if rem == 0 {
			val = tr.Uint64()
			rem = 8
		}

		buf[n] = byte(val)
		val >>= 8
		rem--
	}
	return
}

func (tr *testRand) Uint64() uint64 {
	rv := tr.cur
	tr.cur++
	return rv
}

func (tr *testRand) Seed(seed uint64) {
	tr.cur = seed
}

// TestFactory provides an RNG for testing purposes that simply increments the
// seed value provided to it.
type TestFactory struct {
	seeder Seeder
}

var _ Factory = &TestFactory{}

// New returns a new RNG for this configuration.
func (tf *TestFactory) New() (Rand, error) {
	tr := &testRand{}
	if err := tf.seeder.Seed(tr); err != nil {
		return nil, err
	}

	return tr, nil
}

// NewTestFactory creates a Goroutine-safe factory for constructing test RNGs
// that start from a seed provided by the specified seeder.
func NewTestFactory(seeder Seeder) *TestFactory {
	return &TestFactory{
		seeder: seeder,
	}
}

// ConstantSeeder seeds RNGs with a fixed value for testing purposes.
type ConstantSeeder struct {
	seed uint64
}

var _ Seeder = &ConstantSeeder{}

// Seed sets the target's seed to the seed of this struct.
func (cs *ConstantSeeder) Seed(target Seedable) error {
	target.Seed(cs.seed)
	return nil
}

// NewConstantSeeder creates a new seeder with the given fixed seed.
func NewConstantSeeder(seed uint64) *ConstantSeeder {
	return &ConstantSeeder{
		seed: seed,
	}
}

// ZeroSeeder seeds RNGs with 0 for testing purposes.
var ZeroSeeder = NewConstantSeeder(0)

// OneSeeder seeds RNGs with 1 for testing purposes.
var OneSeeder = NewConstantSeeder(1)

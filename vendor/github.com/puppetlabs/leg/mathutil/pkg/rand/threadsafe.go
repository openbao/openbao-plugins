package rand

import "sync"

// LockableRand is a specialization of Rand that allows implementations to
// choose their own behavior to become Goroutine-safe.
type LockableRand interface {
	Rand

	// ThreadSafe returns a new RNG with the same characteristics as this RNG,
	// but that is also safe to use across Goroutines.
	ThreadSafe() Rand
}

type mutexGuardedRand struct {
	delegate Rand
	mut      sync.Mutex
}

var _ LockableRand = &mutexGuardedRand{}

func (mgr *mutexGuardedRand) Read(buf []byte) (int, error) {
	mgr.mut.Lock()
	defer mgr.mut.Unlock()
	return mgr.delegate.Read(buf)
}

func (mgr *mutexGuardedRand) ThreadSafe() Rand {
	return mgr
}

type mutexGuardedDiscreteRand struct {
	delegate DiscreteRand
	mut      sync.Mutex
}

var _ DiscreteRand = &mutexGuardedDiscreteRand{}
var _ LockableRand = &mutexGuardedDiscreteRand{}

func (mgdr *mutexGuardedDiscreteRand) Read(buf []byte) (int, error) {
	mgdr.mut.Lock()
	defer mgdr.mut.Unlock()
	return mgdr.delegate.Read(buf)
}

func (mgdr *mutexGuardedDiscreteRand) Uint64() uint64 {
	mgdr.mut.Lock()
	defer mgdr.mut.Unlock()
	return mgdr.delegate.Uint64()
}

func (mgdr *mutexGuardedDiscreteRand) ThreadSafe() Rand {
	return mgdr
}

// ThreadSafe makes the given RNG safe to use across Goroutines.
func ThreadSafe(rng Rand) Rand {
	switch t := rng.(type) {
	case LockableRand:
		return t
	case DiscreteRand:
		return &mutexGuardedDiscreteRand{
			delegate: t,
		}
	default:
		return &mutexGuardedRand{
			delegate: t,
		}
	}
}

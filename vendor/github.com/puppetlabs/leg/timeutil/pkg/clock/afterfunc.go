package clock

import (
	"sync"
	"time"
)

type funcTimer struct {
	delegate Timer
	mut      sync.Mutex
	cancel   chan struct{}
	fn       func()
}

var _ Timer = &funcTimer{}

func (ft *funcTimer) C() <-chan time.Time {
	return nil
}

func (ft *funcTimer) Stop() bool {
	r := ft.delegate.Stop()
	if r {
		ft.mut.Lock()
		defer ft.mut.Unlock()

		close(ft.cancel)
		ft.cancel = nil
	}

	return r
}

func (ft *funcTimer) Reset(d time.Duration) bool {
	ft.delegate.Reset(d)
	return ft.schedule()
}

func (ft *funcTimer) schedule() bool {
	ft.mut.Lock()
	defer ft.mut.Unlock()

	if ft.cancel != nil {
		return true
	}

	ch := make(chan struct{})
	go func() {
		select {
		case <-ft.delegate.C():
			ft.mut.Lock()
			ft.cancel = nil
			ft.mut.Unlock()
			ft.fn()
		case <-ch:
		}
	}()

	ft.cancel = ch
	return false
}

// AfterFunc abstracts time.AfterFunc to use a Clock.
//
// Like the time package, the returned Timer will never report a time on its
// channel. The semantics of the Stop and Reset methods are also the same.
func AfterFunc(c Clock, d time.Duration, fn func()) Timer {
	if c == RealClock {
		return &realTimer{time.AfterFunc(d, fn)}
	}
	ft := &funcTimer{
		delegate: c.NewTimer(d),
		fn:       fn,
	}
	ft.schedule()
	return ft
}

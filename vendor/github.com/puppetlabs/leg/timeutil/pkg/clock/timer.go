package clock

import "time"

// TimerCallbackFunc is the callback function for a timer.
type TimerCallbackFunc func(d time.Duration)

type timerCallback struct {
	Timer
	fn TimerCallbackFunc
}

func (tc *timerCallback) Reset(d time.Duration) (r bool) {
	r = tc.Timer.Reset(d)
	tc.fn(d)
	return
}

type timerCallbackClock struct {
	Clock
	fn TimerCallbackFunc
}

func (tcc *timerCallbackClock) NewTimer(d time.Duration) (r Timer) {
	r = &timerCallback{
		Timer: tcc.Clock.NewTimer(d),
		fn:    tcc.fn,
	}
	tcc.fn(d)
	return
}

// NewTimerCallbackClock makes a clock that calls a particular function just
// after NewTimer() or a timer's Reset() is called.
func NewTimerCallbackClock(delegate Clock, fn TimerCallbackFunc) Clock {
	return &timerCallbackClock{
		Clock: delegate,
		fn:    fn,
	}
}

package clock

import "time"

type realTimer struct{ *time.Timer }

func (rt realTimer) C() <-chan time.Time { return rt.Timer.C }

type realTicker struct{ *time.Ticker }

func (rt realTicker) C() <-chan time.Time { return rt.Ticker.C }

type realClock struct{}

func (realClock) Now() time.Time                         { return time.Now() }
func (realClock) Since(t time.Time) time.Duration        { return time.Since(t) }
func (realClock) After(d time.Duration) <-chan time.Time { return time.After(d) }
func (realClock) NewTimer(d time.Duration) Timer         { return &realTimer{time.NewTimer(d)} }
func (realClock) Sleep(d time.Duration)                  { time.Sleep(d) }
func (realClock) NewTicker(d time.Duration) Ticker       { return &realTicker{time.NewTicker(d)} }

// RealClock is the system clock. It delegates to the time package.
var RealClock Clock = &realClock{}

// Package clock defines interfaces for interacting with the passage of time.
//
// For testing, it is sometimes appropriate to inject a mock clock into a
// function to control it, for example, when it uses a timer to make a decision.
//
// This package's interface mirrors that of the Kubernetes API machinery clock
// interface. We encourage you to use their mocks when possible. You can find an
// easy-to-use adapter in the package clock/k8sext.
package clock

import (
	"time"
)

// PassiveClock is a clock that provides timing information but not the ability
// to schedule events for the future.
type PassiveClock interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

// Ticker is an interface that allows for abstraction of time.Ticker.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// Timer is an interface that allows for abstraction of time.Timer.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
	Reset(d time.Duration) bool
}

// Clock is a clock that provides both timing information and the ability to
// schedule.
type Clock interface {
	PassiveClock
	After(time.Duration) <-chan time.Time
	NewTimer(time.Duration) Timer
	Sleep(time.Duration)
	NewTicker(time.Duration) Ticker
}

// Package k8sext provides adapters for the Kubernetes API machinery clock
// implementation to this library.
package k8sext

import (
	"time"

	"github.com/puppetlabs/leg/timeutil/pkg/clock"
	k8sclock "k8s.io/utils/clock"
)

type k8sClock struct{ k8sclock.WithTicker }

func (kc *k8sClock) NewTimer(d time.Duration) clock.Timer {
	t := kc.WithTicker.NewTimer(d)
	kc.WithTicker.Sleep(0)
	return t
}

func (kc *k8sClock) NewTicker(d time.Duration) clock.Ticker {
	t := kc.WithTicker.NewTicker(d)
	kc.WithTicker.Sleep(0)
	return t
}

func NewClock(delegate k8sclock.WithTicker) clock.Clock {
	return &k8sClock{delegate}
}

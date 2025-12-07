package noop

import "github.com/puppetlabs/leg/instrumentation/alerts/trackers"

type NoOp struct{}

func (NoOp) NewCapturer() trackers.Capturer {
	return &Capturer{}
}

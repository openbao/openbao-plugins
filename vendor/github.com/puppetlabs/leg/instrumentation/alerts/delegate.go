package alerts

import "github.com/puppetlabs/leg/instrumentation/alerts/trackers"

type Delegate interface {
	NewCapturer() trackers.Capturer
}

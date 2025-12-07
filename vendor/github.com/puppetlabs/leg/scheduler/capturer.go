package scheduler

import (
	"github.com/puppetlabs/leg/instrumentation/alerts"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

var defaultCapturer = alerts.NewAlerts(alerts.NoDelegate, alerts.Options{}).NewCapturer()

func coalesceCapturer(candidate trackers.Capturer) trackers.Capturer {
	if candidate == nil {
		return defaultCapturer
	}

	return candidate
}

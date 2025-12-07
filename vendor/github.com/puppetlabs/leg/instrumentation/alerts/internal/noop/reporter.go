package noop

import (
	"context"

	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

type Reporter struct{}

func (r Reporter) WithNewTrace() trackers.Reporter {
	return r
}

func (r Reporter) WithTrace(t *trackers.Trace) trackers.Reporter {
	return r
}

func (r Reporter) WithTags(tags ...trackers.Tag) trackers.Reporter {
	return r
}

func (r Reporter) AsWarning() trackers.Reporter {
	return r
}

func (r Reporter) Report(ctx context.Context) <-chan error {
	ch := make(chan error, 1)
	ch <- nil
	return ch
}

func (r Reporter) ReportSync(ctx context.Context) error {
	return <-r.Report(ctx)
}

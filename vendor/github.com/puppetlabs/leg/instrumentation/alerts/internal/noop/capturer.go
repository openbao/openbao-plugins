package noop

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

type Capturer struct{}

func (c Capturer) WithNewTrace() trackers.Capturer {
	return c
}

func (c Capturer) WithAppPackages(packages []string) trackers.Capturer {
	return c
}

func (c Capturer) WithUser(u trackers.User) trackers.Capturer {
	return c
}

func (c Capturer) WithTags(tags ...trackers.Tag) trackers.Capturer {
	return c
}

func (c Capturer) Try(ctx context.Context, fn func(ctx context.Context)) (rv interface{}) {
	defer func() {
		rv = recover()
		if nil != rv {
			debug.PrintStack()
		}
	}()

	fn(ctx)
	return nil
}

func (c Capturer) Capture(err error) trackers.Reporter {
	return &Reporter{}
}

func (c Capturer) CaptureMessage(message string) trackers.Reporter {
	return c.Capture(fmt.Errorf(message))
}

func (c Capturer) Middleware() trackers.Middleware {
	return &Middleware{c: &c}
}

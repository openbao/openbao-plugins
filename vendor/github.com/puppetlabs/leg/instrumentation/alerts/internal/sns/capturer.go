package sns

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

type Capturer struct {
	arn         string
	sopts       session.Options
	newTrace    bool
	appPackages []string
	user        *trackers.User
	tags        []trackers.Tag
}

func (c Capturer) WithNewTrace() trackers.Capturer {
	return &Capturer{
		arn:         c.arn,
		sopts:       c.sopts,
		newTrace:    true,
		appPackages: append([]string{}, c.appPackages...),
		user:        c.user,
		tags:        append([]trackers.Tag{}, c.tags...),
	}
}

func (c Capturer) WithAppPackages(packages []string) trackers.Capturer {
	return &Capturer{
		arn:         c.arn,
		sopts:       c.sopts,
		newTrace:    c.newTrace,
		appPackages: append(append([]string{}, c.appPackages...), packages...),
		user:        c.user,
		tags:        append([]trackers.Tag{}, c.tags...),
	}
}

func (c Capturer) withUser(u trackers.User) *Capturer {
	return &Capturer{
		arn:         c.arn,
		sopts:       c.sopts,
		newTrace:    c.newTrace,
		appPackages: append([]string{}, c.appPackages...),
		user:        &u,
		tags:        append([]trackers.Tag{}, c.tags...),
	}
}

func (c Capturer) WithUser(u trackers.User) trackers.Capturer {
	return c.withUser(u)
}

func (c Capturer) withTags(tags []trackers.Tag) *Capturer {
	return &Capturer{
		arn:         c.arn,
		sopts:       c.sopts,
		newTrace:    c.newTrace,
		appPackages: append([]string{}, c.appPackages...),
		user:        c.user,
		tags:        append(append([]trackers.Tag{}, c.tags...), tags...),
	}
}

func (c Capturer) WithTags(tags ...trackers.Tag) trackers.Capturer {
	return c.withTags(tags)
}

func (c *Capturer) Try(ctx context.Context, fn func(ctx context.Context)) (rv interface{}) {
	ctx = trackers.NewContextWithCapturer(ctx, c)

	defer func() {
		var reporter trackers.Reporter

		rv = recover()
		switch rvt := rv.(type) {
		case nil:
			return
		case error:
			reporter = c.Capture(rvt)
		default:
			reporter = c.CaptureMessage(fmt.Sprint(rvt))
		}

		reporter.Report(ctx)
	}()

	fn(ctx)
	return
}

func (c *Capturer) captureWithStack(err error, skip int) trackers.Reporter {
	return &Reporter{
		c:     c,
		err:   err,
		trace: c.newTrace,
		fs:    trackers.NewTrace(skip + 1),
	}
}

func (c *Capturer) Capture(err error) trackers.Reporter {
	return c.captureWithStack(err, 1)
}

func (c Capturer) CaptureMessage(message string) trackers.Reporter {
	return c.captureWithStack(fmt.Errorf(message), 1)
}

func (c *Capturer) Middleware() trackers.Middleware {
	return &Middleware{
		c: c,
	}
}

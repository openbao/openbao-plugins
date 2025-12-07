package alerts

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/puppetlabs/leg/instrumentation/alerts/internal/noop"
	"github.com/puppetlabs/leg/instrumentation/alerts/internal/passthrough"
	"github.com/puppetlabs/leg/instrumentation/alerts/internal/sentry"
	"github.com/puppetlabs/leg/instrumentation/alerts/internal/sns"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
	"github.com/puppetlabs/leg/instrumentation/errors"
)

type Options struct {
	Environment string
	Version     string
}

type DelegateFunc func(opts Options) Delegate

func NoDelegate(opts Options) Delegate {
	return &noop.NoOp{}
}

func DelegateToPassthrough() (DelegateFunc, errors.Error) {
	b, err := passthrough.NewBuilder()
	if err != nil {
		return nil, err
	}

	fn := func(opts Options) Delegate {
		return b.WithEnvironment(opts.Environment).
			WithRelease(opts.Version).
			Build()
	}
	return fn, nil
}

func DelegateToSNS(arn string, sopts session.Options) (DelegateFunc, errors.Error) {
	b, err := sns.NewBuilder(arn, sopts)
	if err != nil {
		return nil, err
	}

	fn := func(opts Options) Delegate {
		return b.WithEnvironment(opts.Environment).
			WithRelease(opts.Version).
			Build()
	}
	return fn, nil
}

func DelegateToSentry(dsn string) (DelegateFunc, errors.Error) {
	b, err := sentry.NewBuilder(dsn)
	if err != nil {
		return nil, err
	}

	fn := func(opts Options) Delegate {
		return b.WithEnvironment(opts.Environment).
			WithRelease(opts.Version).
			Build()
	}
	return fn, nil
}

type Alerts struct {
	delegate Delegate
}

func (a *Alerts) NewCapturer() trackers.Capturer {
	return a.delegate.NewCapturer()
}

func NewAlerts(fn DelegateFunc, opts Options) *Alerts {
	return &Alerts{
		delegate: fn(opts),
	}
}

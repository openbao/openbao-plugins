package sentry

import (
	raven "github.com/getsentry/raven-go"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
	"github.com/puppetlabs/leg/instrumentation/errors"
)

type Sentry struct {
	client *raven.Client
}

func (s Sentry) NewCapturer() trackers.Capturer {
	return &Capturer{
		client: s.client,
	}
}

type Builder struct {
	client *raven.Client
}

func (b *Builder) WithEnvironment(environment string) *Builder {
	if environment != "" {
		b.client.SetEnvironment(environment)
	}

	return b
}

func (b *Builder) WithRelease(release string) *Builder {
	if release != "" {
		b.client.SetRelease(release)
	}

	return b
}

func (b *Builder) Build() *Sentry {
	return &Sentry{
		client: b.client,
	}
}

func NewBuilder(dsn string) (*Builder, errors.Error) {
	client, err := raven.New(dsn)
	if err != nil {
		return nil, errors.NewAlertsSentryInitializationError().WithCause(err)
	}

	b := &Builder{
		client: client,
	}
	return b, nil
}

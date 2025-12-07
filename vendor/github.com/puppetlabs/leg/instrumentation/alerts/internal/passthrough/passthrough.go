package passthrough

import (
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
	"github.com/puppetlabs/leg/instrumentation/errors"
)

type Passthrough struct {
}

func (p Passthrough) NewCapturer() trackers.Capturer {
	return &Capturer{}
}

type Builder struct {
}

func (b *Builder) WithEnvironment(environment string) *Builder {
	return b
}

func (b *Builder) WithRelease(release string) *Builder {
	return b
}

func (b *Builder) Build() *Passthrough {
	return &Passthrough{}
}

func NewBuilder() (*Builder, errors.Error) {
	b := &Builder{}
	return b, nil
}

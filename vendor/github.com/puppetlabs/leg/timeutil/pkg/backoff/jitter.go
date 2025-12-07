package backoff

import (
	"time"

	"github.com/puppetlabs/leg/mathutil/pkg/rand"
)

type jitterRule struct {
	rng rand.Rand
	pct float64
}

func (jr *jitterRule) ApplyBefore() (bool, time.Duration, error) {
	return true, 0, nil
}

func (jr *jitterRule) ApplyAfter(initial time.Duration) (time.Duration, error) {
	if initial == 0 {
		return initial, nil
	}

	rv, err := rand.Float64Between(jr.rng, float64(initial)*(1-jr.pct), float64(initial))
	if err != nil {
		return 0, err
	}

	return time.Duration(rv), nil
}

type jitterRuleFactory struct {
	rf  rand.Factory
	pct float64
}

func (jrf *jitterRuleFactory) New() (Rule, error) {
	rng, err := jrf.rf.New()
	if err != nil {
		return nil, err
	}

	return &jitterRule{
		rng: rng,
		pct: jrf.pct,
	}, nil
}

// JitterOptions allow for customization of the underlying jitter algorithm.
type JitterOptions struct {
	RandFactory rand.Factory
}

// JitterOption is a setter for one or more jitter options.
type JitterOption interface {
	// ApplyToJitterOptions copies the configuration of this option to the given
	// jitter options.
	ApplyToJitterOptions(target *JitterOptions)
}

// ApplyOptions runs each of the given options against this jitter options
// struct.
func (o *JitterOptions) ApplyOptions(opts []JitterOption) {
	for _, opt := range opts {
		opt.ApplyToJitterOptions(o)
	}
}

// JitterOptionFunc allows a function to be used as a jitter option.
type JitterOptionFunc func(target *JitterOptions)

var _ JitterOption = JitterOptionFunc(nil)

// ApplyToJitterOptions copies the configuration of this option to the given
// jitter options.
func (jof JitterOptionFunc) ApplyToJitterOptions(target *JitterOptions) {
	jof(target)
}

// JitterWithRandFactory returns a jitter option that sets the random number
// generator source for a jitter factory instance.
func JitterWithRandFactory(rf rand.Factory) JitterOption {
	return JitterOptionFunc(func(target *JitterOptions) {
		target.RandFactory = rf
	})
}

// Jitter creates a rule factory that applies a random jitter to a portion of
// the duration. The percentage to apply jitter to should be expressed as an
// decimal number between 0 and 1, inclusive, where 1 means that the entire
// duration is subject to jitter.
func Jitter(pct float64, opts ...JitterOption) RuleFactory {
	o := &JitterOptions{
		RandFactory: rand.DefaultFactory,
	}
	o.ApplyOptions(opts)

	return &jitterRuleFactory{
		rf:  o.RandFactory,
		pct: pct,
	}
}

// EqualJitter creates a rule factory that applies a random jitter to half of
// the duration.
func EqualJitter(opts ...JitterOption) RuleFactory {
	return Jitter(0.5, opts...)
}

// FullJitter creates a rule factory that applies a random jitter to the entire
// duration.
func FullJitter(opts ...JitterOption) RuleFactory {
	return Jitter(1, opts...)
}

package backoff

import (
	"context"
	"math"
	"time"

	"github.com/puppetlabs/leg/mathutil/pkg/rand"
)

type exponentialGenerator struct {
	n       int64
	initial time.Duration
	factor  float64
}

func (eg *exponentialGenerator) Next(ctx context.Context) (time.Duration, error) {
	var exp float64
	switch eg.factor {
	case 2.0:
		exp = math.Exp2(float64(eg.n))
	default:
		exp = math.Pow(eg.factor, float64(eg.n))
	}
	if math.IsInf(exp, 0) || exp > (math.MaxInt64-(1<<9)) {
		return time.Duration(math.MaxInt64), nil
	}

	if eg.n < math.MaxInt64 {
		eg.n++
	}

	return time.Duration(mul64Checked(int64(eg.initial), int64(exp))), nil
}

type exponentialGeneratorFactory struct {
	initial time.Duration
	factor  float64
}

func (egf *exponentialGeneratorFactory) New() (Generator, error) {
	return &exponentialGenerator{
		initial: egf.initial,
		factor:  egf.factor,
	}, nil
}

// Exponential creates a generator factory for backoffs that start at the given
// initial duration and increase multiplicatively by the specified factor.
func Exponential(initial time.Duration, factor float64) GeneratorFactory {
	return &exponentialGeneratorFactory{
		initial: initial,
		factor:  factor,
	}
}

type decorrelatedExponentialGenerator struct {
	cur     int64
	rng     rand.Rand
	initial time.Duration
	factor  float64
	rule    Rule
}

var _ RuleInjector = &decorrelatedExponentialGenerator{}

func (deg *decorrelatedExponentialGenerator) Next(ctx context.Context) (next time.Duration, err error) {
	generate := true
	if deg.rule != nil {
		generate, next, err = deg.rule.ApplyBefore()
		if err != nil {
			return 0, err
		}
	}

	if generate {
		rv, err := rand.Float64Between(deg.rng, float64(deg.initial), float64(deg.cur)*deg.factor)
		if err != nil {
			return 0, err
		}

		if math.IsInf(rv, 0) || rv > (math.MaxInt64-(1<<9)) {
			next = time.Duration(math.MaxInt64)
		} else {
			next = time.Duration(rv)
		}
	}

	// We want to make sure that we clamp this value according to any rules that
	// need to be applied like a maximum duration.
	if deg.rule != nil {
		next, err = deg.rule.ApplyAfter(next)
		if err != nil {
			return 0, err
		}
	}

	if next < deg.initial {
		deg.cur = int64(deg.initial)
	} else {
		deg.cur = int64(next)
	}

	return
}

func (deg *decorrelatedExponentialGenerator) InjectRule(r Rule) {
	deg.rule = r
}

type decorrelatedExponentialGeneratorFactory struct {
	rf      rand.Factory
	initial time.Duration
	factor  float64
}

func (degf *decorrelatedExponentialGeneratorFactory) New() (Generator, error) {
	rng, err := degf.rf.New()
	if err != nil {
		return nil, err
	}

	return &decorrelatedExponentialGenerator{
		cur:     int64(degf.initial),
		rng:     rng,
		initial: degf.initial,
		factor:  degf.factor,
	}, nil
}

// DecorrelatedExponentialOptions allow for customization of the underlying
// decorrelated exponential algorithm.
type DecorrelatedExponentialOptions struct {
	RandFactory rand.Factory
}

// DecorrelatedExponentialOption is a setter for one or more decorrelated
// exponential options.
type DecorrelatedExponentialOption interface {
	// ApplyToDecorrelatedExponentialOptions copies the configuration of this
	// option to the given decorrelated exponential options.
	ApplyToDecorrelatedExponentialOptions(target *DecorrelatedExponentialOptions)
}

// ApplyOptions runs each of the given options against this decorrelated
// exponential options struct.
func (o *DecorrelatedExponentialOptions) ApplyOptions(opts []DecorrelatedExponentialOption) {
	for _, opt := range opts {
		opt.ApplyToDecorrelatedExponentialOptions(o)
	}
}

// DecorrelatedExponentialOptionFunc allows a function to be used as a
// decorrelated exponential option.
type DecorrelatedExponentialOptionFunc func(target *DecorrelatedExponentialOptions)

var _ DecorrelatedExponentialOption = DecorrelatedExponentialOptionFunc(nil)

// ApplyToDecorrelatedExponentialOptions copies the configuration of this option
// to the given decorrelated exponential options.
func (deof DecorrelatedExponentialOptionFunc) ApplyToDecorrelatedExponentialOptions(target *DecorrelatedExponentialOptions) {
	deof(target)
}

// DecorrelatedExponentialWithRandFactory returns a decorrelated exponential
// option that sets the random number generator source for a decorrelated
// exponential factory instance.
func DecorrelatedExponentialWithRandFactory(rf rand.Factory) DecorrelatedExponentialOption {
	return DecorrelatedExponentialOptionFunc(func(target *DecorrelatedExponentialOptions) {
		target.RandFactory = rf
	})
}

// DecorrelatedExponential creates a generator factory for a specialized type of
// exponential backoff algorithm that applies a random jitter to the factor
// instead of to the value produced.
func DecorrelatedExponential(initial time.Duration, factor float64, opts ...DecorrelatedExponentialOption) GeneratorFactory {
	o := &DecorrelatedExponentialOptions{
		RandFactory: rand.DefaultFactory,
	}
	o.ApplyOptions(opts)

	return &decorrelatedExponentialGeneratorFactory{
		rf:      o.RandFactory,
		initial: initial,
		factor:  factor,
	}
}

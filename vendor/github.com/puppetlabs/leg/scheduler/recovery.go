package scheduler

import (
	"context"
	"reflect"
	"time"

	"github.com/puppetlabs/leg/timeutil/pkg/backoff"
	"github.com/puppetlabs/leg/timeutil/pkg/clock"
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
)

// DefaultRecoveryDescriptorBackoffFactory is an exponential backoff starting at
// 5 milliseconds with a factor of 2, a 10 second cap, and full jitter using the
// default RNG. It automatically resets the backoff after 30 seconds of
// inactivity.
var DefaultRecoveryDescriptorBackoffFactory = backoff.Build(
	backoff.ResetAfter(
		backoff.Build(
			backoff.Exponential(5*time.Millisecond, 2.0),
			backoff.NonSliding,
		),
		30*time.Second,
	),
	backoff.MaxBound(10*time.Second),
	backoff.FullJitter(),
)

// RecoveryDescriptorOptions contains fields that allow backoff and retry
// parameters to be set.
type RecoveryDescriptorOptions struct {
	// BackoffFactory is the backoff algorithm to use when a guarded descriptor
	// fails. If not specified, a sensibile default
	// (DefaultRecoveryDescriptorBackoffFactory) is used.
	BackoffFactory *backoff.Factory

	// Clock is the clock implementation used to perform the backoff.
	Clock clock.Clock
}

// RecoveryDescriptorOption is a setter for one or more recovery descriptor
// options.
type RecoveryDescriptorOption interface {
	// ApplyToRecoveryDescriptorOptions configures the specified recovery
	// descriptor options for this option.
	ApplyToRecoveryDescriptorOptions(target *RecoveryDescriptorOptions)
}

// ApplyOptions runs each of the given options against this options struct.
func (o *RecoveryDescriptorOptions) ApplyOptions(opts []RecoveryDescriptorOption) {
	for _, opt := range opts {
		opt.ApplyToRecoveryDescriptorOptions(o)
	}
}

// RecoveryDescriptorOptionFunc allows a function to be used as a recovery
// descriptor option.
type RecoveryDescriptorOptionFunc func(target *RecoveryDescriptorOptions)

var _ RecoveryDescriptorOption = RecoveryDescriptorOptionFunc(nil)

// ApplyToRecoveryDescriptorOptions configures the specified recovery descriptor
// options by calling this function.
func (rdof RecoveryDescriptorOptionFunc) ApplyToRecoveryDescriptorOptions(target *RecoveryDescriptorOptions) {
	rdof(target)
}

// RecoveryDescriptorWithBackoffFactory changes the backoff algorithm to the
// specified one.
func RecoveryDescriptorWithBackoffFactory(bf *backoff.Factory) RecoveryDescriptorOption {
	return RecoveryDescriptorOptionFunc(func(target *RecoveryDescriptorOptions) {
		target.BackoffFactory = bf
	})
}

// RecoveryDescriptorWithClock changes the backoff clock to the specified one.
func RecoveryDescriptorWithClock(c clock.Clock) RecoveryDescriptorOption {
	return RecoveryDescriptorOptionFunc(func(target *RecoveryDescriptorOptions) {
		target.Clock = c
	})
}

// RecoveryDescriptor wraps a given descriptor so that it restarts if the
// descriptor itself fails. This is useful for descriptors that work off of
// external information (APIs, events, etc.).
type RecoveryDescriptor struct {
	delegate       Descriptor
	backoffFactory *backoff.Factory
	clock          clock.Clock
}

var _ Descriptor = &RecoveryDescriptor{}

// Run delegates work to another descriptor, catching any errors are restarting
// the descriptor immediately if an error occurs. It might return a max retries error.
// It only terminates when the context is done or the max retries have been exceeded.
func (rd *RecoveryDescriptor) Run(ctx context.Context, pc chan<- Process) error {
	waitOptions := []retry.WaitOption{retry.WithBackoffFactory(rd.backoffFactory)}
	if rd.clock != nil {
		waitOptions = append(waitOptions, retry.WithClock(rd.clock))
	}

	return retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		err := rd.delegate.Run(ctx, pc)

		// We'll eat the context error here as we don't want to propagate it.
		select {
		case <-ctx.Done():
			return true, err
		default:
		}

		if err != nil {
			log(ctx).Warn("recovering failing descriptor", "descriptor", reflect.TypeOf(rd.delegate).String(), "error", err)
		}

		return err == nil, err
	}, waitOptions...)
}

// NewRecoveryDescriptor creates a new recovering descriptor wrapping the given
// delegate descriptor. Default backoff and retry parameters will be used.
func NewRecoveryDescriptor(delegate Descriptor, opts ...RecoveryDescriptorOption) *RecoveryDescriptor {
	o := &RecoveryDescriptorOptions{
		BackoffFactory: DefaultRecoveryDescriptorBackoffFactory,
	}
	o.ApplyOptions(opts)

	return &RecoveryDescriptor{
		delegate:       delegate,
		backoffFactory: o.BackoffFactory,
		clock:          o.Clock,
	}
}

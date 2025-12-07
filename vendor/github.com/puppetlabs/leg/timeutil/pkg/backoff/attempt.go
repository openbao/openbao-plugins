package backoff

import (
	"fmt"
	"math"
	"time"
)

// MaxAttemptsReachedError is the error produced by the MaxAttempts rule when
// the limit has been reached.
type MaxAttemptsReachedError struct {
	N uint64
}

func (e *MaxAttemptsReachedError) Error() string {
	return fmt.Sprintf("maximum attempts of %d reached", e.N)
}

type maxAttemptsRule struct {
	cur uint64
	max uint64
}

func (mar *maxAttemptsRule) ApplyBefore() (bool, time.Duration, error) {
	if mar.cur >= mar.max {
		return false, 0, &MaxAttemptsReachedError{N: mar.max}
	}

	if mar.cur < math.MaxUint64 {
		mar.cur++
	}
	return true, 0, nil
}

func (mar *maxAttemptsRule) ApplyAfter(initial time.Duration) (time.Duration, error) {
	return initial, nil
}

type maxAttemptsRuleFactory struct {
	max uint64
}

func (marf *maxAttemptsRuleFactory) New() (Rule, error) {
	return &maxAttemptsRule{
		max: marf.max,
	}, nil
}

// MaxAttempts creates a rule factory that limits the number of possible
// attempts made, producing an error every time Next() is called after that.
func MaxAttempts(max uint64) RuleFactory {
	return &maxAttemptsRuleFactory{
		max: max,
	}
}

// MaxRetries for a given maximum count is the same as calling
// MaxAttempts(max+1). Sometimes it's easier to think in terms of retries,
// though.
func MaxRetries(max uint64) RuleFactory {
	return MaxAttempts(max + 1)
}

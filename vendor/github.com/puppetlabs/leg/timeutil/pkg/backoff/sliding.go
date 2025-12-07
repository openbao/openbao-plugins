package backoff

import (
	"time"
)

type nonSlidingRule struct {
	once bool
}

func (nsr *nonSlidingRule) ApplyBefore() (bool, time.Duration, error) {
	return nsr.once, 0, nil
}

func (nsr *nonSlidingRule) ApplyAfter(initial time.Duration) (time.Duration, error) {
	if !nsr.once {
		nsr.once = true
		return 0, nil
	}

	return initial, nil
}

type nonSlidingRuleFactory struct{}

func (*nonSlidingRuleFactory) New() (Rule, error) {
	return &nonSlidingRule{}, nil
}

// NonSliding causes the first iteration of a backoff to be zero instead of an
// initial backoff amount determined by the generator.
var NonSliding RuleFactory = &nonSlidingRuleFactory{}

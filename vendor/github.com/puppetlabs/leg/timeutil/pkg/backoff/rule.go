package backoff

import (
	"time"
)

// Rule adjusts a backoff duration by applying additional logic like jitter or
// bounding.
type Rule interface {
	// ApplyBefore allows this rule to intervene before the generator produces a
	// new duration.
	ApplyBefore() (bool, time.Duration, error)

	// ApplyAfter modifies the duration according this rule.
	ApplyAfter(initial time.Duration) (time.Duration, error)
}

// RuleFactory is a Goroutine-safe factory for creating rules that apply to a
// particular algorithm.
type RuleFactory interface {
	New() (Rule, error)
}

// Rules encodes a list of rules as a single rule.
type Rules []Rule

var _ Rule = Rules{}

// ApplyBefore iterates through the rule set and applies each rule in order. If
// any rule returns false, no further rules are tested.
func (r Rules) ApplyBefore() (bool, time.Duration, error) {
	for _, rule := range r {
		generate, next, err := rule.ApplyBefore()
		if err != nil {
			return false, 0, err
		} else if !generate {
			return false, next, nil
		}
	}

	return true, 0, nil
}

// ApplyAfter iterates through the rule set and applies each rule in order.
func (r Rules) ApplyAfter(initial time.Duration) (time.Duration, error) {
	cur := initial
	for _, rule := range r {
		next, err := rule.ApplyAfter(cur)
		if err != nil {
			return 0, err
		}
		cur = next
	}
	return cur, nil
}

// RuleInjector allows a backoff to customize rule application, for example, if
// it needs to use the result of the rule application as part of its algorithm.
type RuleInjector interface {
	InjectRule(r Rule)
}

package errmark

import (
	"errors"
	"reflect"
)

// Rule tests an error against an arbitrary condition.
//
// It is useful for deciding when an error should be marked with a given Marker.
type Rule interface {
	Matches(err error) bool
}

// RuleFunc allows a function to be used as a Rule.
type RuleFunc func(err error) bool

var _ Rule = RuleFunc(nil)

func (rf RuleFunc) Matches(err error) bool {
	return rf(err)
}

// RuleExact matches an error if it is exactly the wanted error.
func RuleExact(want error) Rule {
	return RuleFunc(func(err error) bool {
		return want == err
	})
}

// RuleIs matches an error if errors.Is() would return true for the wanted
// error.
func RuleIs(want error) Rule {
	return RuleFunc(func(err error) bool {
		return errors.Is(err, want)
	})
}

// RuleType matches an error if errors.As() would return true for a target of a
// pointer to the type given.
func RuleType(want interface{}) Rule {
	// This will give us a pointer to whatever type is wanted.
	candidate := reflect.New(reflect.ValueOf(want).Type()).Interface()
	return RuleFunc(func(err error) bool {
		return errors.As(err, candidate)
	})
}

// RuleAll applies several rules in sequence. If one of the rules returns false,
// this rule also returns false. Otherwise, this rule returns true.
func RuleAll(rules ...Rule) Rule {
	return RuleFunc(func(err error) bool {
		for _, rule := range rules {
			if !rule.Matches(err) {
				return false
			}
		}

		return true
	})
}

// RuleAlways succeeds against every error.
var RuleAlways = RuleAll()

// RuleAny applies several rules in sequence. If one of the rules returns true,
// this rule also returns true. Otherwise, this rule returns false.
func RuleAny(rules ...Rule) Rule {
	return RuleFunc(func(err error) bool {
		for _, rule := range rules {
			if rule.Matches(err) {
				return true
			}
		}

		return false
	})
}

// RuleNever fails against every error.
var RuleNever = RuleAny()

// RuleNot inverts the result of a delegate rule.
func RuleNot(delegate Rule) Rule {
	return RuleFunc(func(err error) bool {
		return !delegate.Matches(err)
	})
}

// RulePredicate evalutes a delegate rule when a predicate is satisfied at the
// time an error is passed to the rule. It returns false if the predicate is not
// satisfied.
func RulePredicate(delegate Rule, when func() bool) Rule {
	return RuleFunc(func(err error) bool {
		if when() {
			return delegate.Matches(err)
		}

		return false
	})
}

// RuleMarked matches an error if the error has been marked with the given
// Marker.
func RuleMarked(m *Marker) Rule {
	return RuleFunc(func(err error) bool {
		return Markers(err).Has(m)
	})
}

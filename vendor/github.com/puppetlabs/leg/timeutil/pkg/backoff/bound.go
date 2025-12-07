package backoff

import "time"

type minBoundRule struct {
	min time.Duration
}

func (mbr *minBoundRule) ApplyBefore() (bool, time.Duration, error) {
	return true, 0, nil
}

func (mbr *minBoundRule) ApplyAfter(initial time.Duration) (time.Duration, error) {
	if initial < mbr.min {
		return mbr.min, nil
	}

	return initial, nil
}

func (mbr *minBoundRule) New() (Rule, error) {
	return mbr, nil
}

// MinBound creates a rule factory that guarantees a duration is never less than
// a specified value.
func MinBound(min time.Duration) RuleFactory {
	return &minBoundRule{
		min: min,
	}
}

type maxBoundRule struct {
	max time.Duration
}

func (mbr *maxBoundRule) ApplyBefore() (bool, time.Duration, error) {
	return true, 0, nil
}

func (mbr *maxBoundRule) ApplyAfter(initial time.Duration) (time.Duration, error) {
	if initial > mbr.max {
		return mbr.max, nil
	}

	return initial, nil
}

func (mbr *maxBoundRule) New() (Rule, error) {
	return mbr, nil
}

// MaxBound creates a rule factory that guarantees a duration is never more than
// a specified value.
func MaxBound(max time.Duration) RuleFactory {
	return &maxBoundRule{
		max: max,
	}
}

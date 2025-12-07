package rand

import "errors"

var (
	// ErrImpossible is returned by discrete functions if an invocation would
	// result in a random number generator faulting because the parameters
	// requested cannot be computed.
	ErrImpossible = errors.New("rand: impossible construction")
)

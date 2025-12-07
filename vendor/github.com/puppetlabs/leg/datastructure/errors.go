package datastructure

import (
	"errors"
)

var (
	// ErrStopIteration causes a ForEach() or ForEachInto() loop to terminate
	// early.
	ErrStopIteration = errors.New("datastructure: stop iteration")

	// ErrInvalidFuncSignature is raised in a panic() if a function passed to a
	// ForEachInto() method does not conform to the expected interface.
	ErrInvalidFuncSignature = errors.New("datastructure: invalid function signature")
)

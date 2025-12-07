package scheduler

import (
	"fmt"

	"github.com/puppetlabs/leg/request"
)

type PanicError struct {
	Cause error
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %+v", e.Cause)
}

func coercePanic(r interface{}) error {
	err, ok := r.(error)
	if !ok {
		err = fmt.Errorf("%+v", r)
	}

	return &PanicError{
		Cause: err,
	}
}

type ProcessError struct {
	Request *request.Request
	Process Process
	Cause   error
}

func (e *ProcessError) Error() string {
	return fmt.Sprintf("process %s (request %s): %+v", e.Process.Description(), e.Request.Identifier, e.Cause)
}

type DescriptorError struct {
	Descriptor Descriptor
	Cause      error
}

func (e *DescriptorError) Error() string {
	return fmt.Sprintf("descriptor %T: %+v", e.Descriptor, e.Cause)
}

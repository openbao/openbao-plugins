package datastructure

import (
	"reflect"
)

func coalesceInvalidToZeroValueOf(v reflect.Value, ifInvalid reflect.Type) reflect.Value {
	if !v.IsValid() {
		v = reflect.Zero(ifInvalid)
	}

	return v
}

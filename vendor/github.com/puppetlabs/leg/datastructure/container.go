// Portions of this file are derived from GoDS, a data structure library for Go.
//
// Copyright (c) 2015, Emir Pasic. All rights reserved.
//
// https://github.com/emirpasic/gods/blob/213367f1ca932600ce530ae11c8a8cc444e3a6da/containers/containers.go
//
// Licensed under a two-clause BSD-style license. A copy of the full license
// document is included in this distribution in the file `container.go.LICENSE`.

package datastructure

// Container represents any type with elements, such as a map, list, or set.
type Container interface {
	// Empty returns true if this container is empty; false otherwise.
	//
	// The result of this function is equivalent to the test Size() == 0, but
	// certain data structures that do not have an O(1) Size() implementation
	// may optimize this function.
	Empty() bool

	// Size returns the number of values in this container.
	Size() int

	// Clear resets this container to its zero-value state.
	Clear()

	// Values returns the values from this container as a slice of interface{}.
	Values() []interface{}

	// ValuesInto inserts the values from this container into the given slice.
	// The type of the into parameter must be a pointer to a slice for which
	// each value must be assignable by the type of every value in this
	// container.
	//
	// If the requirements for the into parameter are not met, this function
	// will panic.
	ValuesInto(into interface{})
}

// Portions of this file are derived from GoDS, a data structure library for Go.
//
// Copyright (c) 2015, Emir Pasic. All rights reserved.
//
// https://github.com/emirpasic/gods/blob/213367f1ca932600ce530ae11c8a8cc444e3a6da/sets/sets.go
//
// Licensed under a two-clause BSD-style license. A copy of the full license
// document is included in this distribution in the file `set.go.LICENSE`.

package datastructure

type SetIterationFunc func(element interface{}) error

// A Set is a well-defined collection of distinct objects.
type Set interface {
	Container

	// Contains returns true if the set contains all of the elements specified,
	// and false otherwise.
	Contains(elements ...interface{}) bool

	// Adds inserts all of the elements specified to the set. If an element
	// already exists in the set, it will not be duplicated.
	Add(elements ...interface{})

	// AddAll inserts the values from the given container to this set. If an
	// element already exists in the set, it will not be duplicated. Equivalent
	// to a set union operation.
	AddAll(other Container)

	// Remove eliminates all of the elements specified from the set.
	Remove(elements ...interface{})

	// RemoveAll removes the elements from the given set from this set.
	// Equivalent to a set difference operation.
	RemoveAll(other Set)

	// ForEach each element in the set and executes the given callback function.
	// If the callback function returns an error, this function will return the
	// same error and immediately stop iteration.
	//
	// To stop iteration without returning an error, return ErrStopIteration.
	ForEach(fn SetIterationFunc) error

	// ForEachInto iterates each element in the set and executes the given
	// callback function, which must be of a type similar to SetIterationFunc,
	// except that the element parameter may be any type assignable by every
	// element in the set.
	//
	// If the requirements for the fn parameter are not met, this function will
	// panic.
	ForEachInto(fn interface{}) error

	// RetailAll removes any elements that are not shared between this set and
	// the given set. Equivalent to a set intersection operation.
	RetainAll(other Set)
}

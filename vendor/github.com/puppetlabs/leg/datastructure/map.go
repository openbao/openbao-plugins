// Portions of this file are derived from GoDS, a data structure library for Go.
//
// Copyright (c) 2015, Emir Pasic. All rights reserved.
//
// https://github.com/emirpasic/gods/blob/52d942a0538c185239fa3737047f297d983ac3e0/maps/maps.go
//
// Licensed under a two-clause BSD-style license. A copy of the full license
// document is included in this distribution in the file `map.go.LICENSE`.

package datastructure

type MapIterationFunc func(key, value interface{}) error

// Map represents a key-to-value mapping, where keys are unique.
type Map interface {
	Container

	// Contains returns true if the given key exists in this map.
	Contains(key interface{}) bool

	// Put adds the given key and value to the map. If the value already exists
	// in the map, the old value is overwritten by the specified value.
	//
	// Returns true if this map already contained an entry for this key, and
	// false otherwise.
	Put(key, value interface{}) bool

	// Get retrieves the value associated with the given key from the map.
	//
	// Returns the value, or nil if the key does not exist in the map, and a
	// boolean indicating whether the key exists.
	Get(key interface{}) (interface{}, bool)

	// GetInto retrieves the value associated with the given key from the map
	// and stores it in the into parameter passed to this function.
	//
	// The into parameter must be a pointer to a type assignable by the stored
	// value. If the given key does not exist in the map, the into parameter is
	// not modified.
	//
	// If the into parameter is not compatible with the stored value, this
	// function will panic.
	//
	// Returns true if the key exists, and false otherwise.
	GetInto(key interface{}, into interface{}) bool

	// Remove eliminates the given key from the map, and returns true if the key
	// existed in the map.
	Remove(key interface{}) bool

	// Keys returns the keys from this map as a slice of interface{}.
	Keys() []interface{}

	// KeysInto inserts the keys from this map into the given slice. The type of
	// the into parameter must be a pointer to a slice for which each value must
	// be assignable by the type of every key in this map.
	//
	// If the requirements for the into parameter are not met, this function
	// will panic.
	KeysInto(into interface{})

	// ForEach iterates each key-value pair in the map and executes the given
	// callback function. If the callback function returns an error, this
	// function will return the same error and immediately stop iteration.
	//
	// To stop iteration without returning an error, return ErrStopIteration.
	ForEach(fn MapIterationFunc) error

	// ForEachInto iterates each key-value pair in the map and executes the
	// given callback function, which must be of a type similar to
	// MapIterationFunc, except that the key and value parameters may be any
	// type assignable by every key and value in the map, respectively.
	//
	// If the requirements for the fn parameter are not met, this function will
	// panic.
	ForEachInto(fn interface{}) error
}

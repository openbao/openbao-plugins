package errmap

import (
	"errors"
	"fmt"
)

// Mapper constructs a new error from a given error.
//
// Generally, a mapped error should provied an Unwrap method to access the
// underlying error unless it is deliberately hiding it (because of an API
// boundary, for example).
type Mapper interface {
	Map(err error) error
}

// MapperFunc allows a function to be used as a Mapper.
type MapperFunc func(err error) error

var _ Mapper = MapperFunc(nil)

func (mf MapperFunc) Map(err error) error {
	return mf(err)
}

// MapApplicator allows an error to control mapping behavior.
//
// An implementor of MapApplicator generally maps a delegate error and then
// determines what action it needs to take. For example, it may choose to merge
// the resulting error with itself, or may opt to take no action at all.
//
// Each error that is mapped by this package is tested to see if it implements
// MapApplicator before Mappers are called.
type MapApplicator interface {
	MapApply(m Mapper) error
}

type mappedError struct {
	base   error
	before []Mapper
	after  []Mapper
}

func (e *mappedError) Error() string {
	return e.Apply().Error()
}

func (e *mappedError) Unwrap() error {
	return errors.Unwrap(e.Apply())
}

func (e *mappedError) Is(target error) bool {
	return errors.Is(e.Apply(), target)
}

func (e *mappedError) As(target interface{}) bool {
	return errors.As(e.Apply(), target)
}

func (e *mappedError) Apply() error {
	err := e.base
	for _, ms := range [][]Mapper{e.before, e.after} {
		for _, m := range ms {
			if mapper, ok := err.(MapApplicator); ok {
				err = mapper.MapApply(m)
			} else {
				err = m.Map(err)
			}
		}
	}
	return err
}

func (e *mappedError) InsertBefore(m Mapper) {
	e.before = append([]Mapper{m}, e.before...)
}

func (e *mappedError) AppendBefore(m Mapper) {
	e.before = append(e.before, m)
}

func (e *mappedError) InsertAfter(m Mapper) {
	e.after = append([]Mapper{m}, e.after...)
}

func (e *mappedError) AppendAfter(m Mapper) {
	e.after = append(e.after, m)
}

func toMappedError(err error) *mappedError {
	e, ok := err.(*mappedError)
	if !ok {
		e = &mappedError{base: err}
	}

	return e
}

// MapFirst causes the given mapper to run at the very beginning of an error
// mapping chain.
func MapFirst(err error, m Mapper) error {
	e := toMappedError(err)
	e.InsertBefore(m)
	return e
}

// MapBefore causes the given mapper to run just before any mappers added by
// calling MapAfter in an error mapping chain.
func MapBefore(err error, m Mapper) error {
	e := toMappedError(err)
	e.AppendBefore(m)
	return e
}

// MapAfter causes the given mapper to run just after any mappers added by
// calling MapBefore in an error mapping chain.
func MapAfter(err error, m Mapper) error {
	e := toMappedError(err)
	e.InsertAfter(m)
	return e
}

// MapLast causes the given mapper to run at the very end of an error mapping
// chain.
func MapLast(err error, m Mapper) error {
	e := toMappedError(err)
	e.AppendAfter(m)
	return e
}

// Apply invokes all of the mappers that have been added to the given error and
// returns the resulting error.
//
// Once Apply has been called for an error, the mapping chain is reset.
func Apply(err error) error {
	if merr, ok := err.(*mappedError); ok {
		return merr.Apply()
	}

	return err
}

// Wrap causes the given message to be prepended to the error at the end of the
// error mapping chain.
func Wrap(err error, message string) error {
	return MapLast(err, MapperFunc(func(err error) error {
		return fmt.Errorf("%s: %w", message, err)
	}))
}

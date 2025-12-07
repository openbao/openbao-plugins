package errmark

import (
	"errors"
	"fmt"
	"strings"

	"github.com/puppetlabs/leg/errmap/pkg/errmap"
)

type markedError struct {
	base    error
	markers *MarkerSet
}

var _ errmap.MapApplicator = &markedError{}

func (e *markedError) Error() string {
	if MarkedShort(e) {
		return e.base.Error()
	}

	return fmt.Sprintf("%s: %v", strings.Join(e.markers.Names(), ": "), e.base)
}

func (e *markedError) MapApply(m errmap.Mapper) error {
	next := m.Map(e.base)

	switch nt := next.(type) {
	case *markedError:
		return &markedError{
			base:    nt.base,
			markers: e.markers.Merge(nt.markers),
		}
	default:
		return &markedError{
			base:    next,
			markers: e.markers,
		}
	}
}

func (e *markedError) Unwrap() error {
	return e.base
}

// MarkIf marks an error with a particular Marker if the given Rule matches the
// error.
func MarkIf(err error, m *Marker, rule Rule) error {
	return errmap.MapBefore(err, errmap.MapperFunc(func(err error) error {
		if Matches(err, RuleAll(RuleNot(RuleMarked(m)), rule)) {
			return &markedError{
				base:    err,
				markers: NewMarkerSet(m),
			}
		}

		return err
	}))
}

// Mark marks an error with a particular Marker.
func Mark(err error, m *Marker) error {
	return MarkIf(err, m, RuleAlways)
}

// Matches returns true if the given Rule matches the error.
func Matches(err error, rule Rule) bool {
	return rule.Matches(errmap.Apply(err))
}

// Marked returns true if the given error has had a particular Marker applied.
func Marked(err error, m *Marker) bool {
	return Matches(err, RuleMarked(m))
}

// Markers returns a set of all the markers that are present on the given error.
func Markers(err error) *MarkerSet {
	merr := &markedError{}
	if errors.As(err, &merr) {
		return merr.markers.Merge(Markers(merr.base))
	}

	return nil
}

// If runs a callback function if the given Rule matches the error.
func If(err error, rule Rule, fn func(err error)) {
	if Matches(err, rule) {
		fn(err)
	}
}

// IfMarked runs a callback function if the error is marked with the given
// Marker.
func IfMarked(err error, m *Marker, fn func(err error)) {
	If(err, RuleMarked(m), fn)
}

// Unless runs a callback function if the given Rule does not match the error.
func Unless(err error, rule Rule, fn func(err error)) {
	If(err, RuleNot(rule), fn)
}

// UnlessMarked runs a callback function if the error is not marked with the
// given Marker.
func UnlessMarked(err error, m *Marker, fn func(err error)) {
	Unless(err, RuleMarked(m), fn)
}

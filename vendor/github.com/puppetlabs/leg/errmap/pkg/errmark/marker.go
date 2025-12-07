package errmark

import "github.com/puppetlabs/leg/datastructure"

// Marker represents a named identifier that logically groups one or more
// arbitrary errors.
//
// Note that Markers are intentionally not comparable by value, that is,
// NewMarker("foo") != NewMarker("foo"). Markers should generally be declared in
// package scope.
type Marker struct {
	name string
}

// Name returns the supplied name of this marker.
func (m *Marker) Name() string {
	return m.name
}

// NewMarker creates a Marker with the given name.
func NewMarker(name string) *Marker {
	return &Marker{name: name}
}

// MarkerSet is a unique collection of Markers.
type MarkerSet struct {
	storage datastructure.Set
}

// Names returns a list of names of each unique Marker in this collection in the
// order they were added to the collection.
func (ms *MarkerSet) Names() []string {
	if ms == nil {
		return nil
	}

	l := make([]string, 0, ms.storage.Size())
	_ = ms.storage.ForEachInto(func(m *Marker) error {
		l = append(l, m.Name())
		return nil
	})
	return l
}

// Merge combines one MarkerSet with another, retaining only unique Markers. A
// new MarkerSet is returned (unless either this MarkerSet or the other
// MarkerSet are empty, in which case the non-empty set is returned directly).
func (ms *MarkerSet) Merge(other *MarkerSet) *MarkerSet {
	switch {
	case other == nil || other.storage.Size() == 0:
		return ms
	case ms == nil || ms.storage.Size() == 0:
		return other
	default:
		union := NewMarkerSet()
		union.storage.AddAll(ms.storage)
		union.storage.AddAll(other.storage)
		return union
	}
}

// Has tests whether this MarkerSet contains the given Marker.
func (ms *MarkerSet) Has(m *Marker) bool {
	if ms == nil {
		return false
	}

	return ms.storage.Contains(m)
}

// NewMarkerSet creates a collection of Markers with the given values after
// deduplicating.
//
// It is valid to work with nil values of *MarkerSet. All methods of the struct
// check for a nil receiver first.
func NewMarkerSet(values ...*Marker) *MarkerSet {
	s := datastructure.NewLinkedHashSet()
	for _, value := range values {
		s.Add(value)
	}

	return &MarkerSet{
		storage: s,
	}
}

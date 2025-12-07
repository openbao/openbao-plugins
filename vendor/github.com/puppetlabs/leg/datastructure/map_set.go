package datastructure

// MapBackedSet is a set that stores its elements as keys in a given map.
type MapBackedSet struct {
	storage Map
}

var mapSetValue = struct{}{}

func (s *MapBackedSet) Contains(elements ...interface{}) bool {
	for _, element := range elements {
		if !s.storage.Contains(element) {
			return false
		}
	}

	return true
}

func (s *MapBackedSet) Add(elements ...interface{}) {
	for _, element := range elements {
		s.storage.Put(element, mapSetValue)
	}
}

func (s *MapBackedSet) AddAll(other Container) {
	s.Add(other.Values()...)
}

func (s *MapBackedSet) Remove(elements ...interface{}) {
	for _, element := range elements {
		s.storage.Remove(element)
	}
}

func (s *MapBackedSet) RemoveAll(other Set) {
	var remove []interface{}

	s.ForEach(func(element interface{}) error {
		if other.Contains(element) {
			remove = append(remove, element)
		}

		return nil
	})

	for _, element := range remove {
		s.Remove(element)
	}
}

func (s *MapBackedSet) Empty() bool {
	return s.storage.Empty()
}

func (s *MapBackedSet) Size() int {
	return s.storage.Size()
}

func (s *MapBackedSet) Clear() {
	s.storage.Clear()
}

func (s *MapBackedSet) Values() []interface{} {
	return s.storage.Keys()
}

func (s *MapBackedSet) ValuesInto(into interface{}) {
	setValuesInto(s, into)
}

func (s *MapBackedSet) ForEach(fn SetIterationFunc) error {
	return s.storage.ForEach(func(key, value interface{}) error {
		return fn(key)
	})
}

func (s *MapBackedSet) ForEachInto(fn interface{}) error {
	return setForEachInto(s, fn)
}

func (s *MapBackedSet) RetainAll(other Set) {
	var remove []interface{}

	s.ForEach(func(element interface{}) error {
		if !other.Contains(element) {
			remove = append(remove, element)
		}

		return nil
	})

	for _, element := range remove {
		s.Remove(element)
	}
}

// NewMapBackedSet creates a new map-backed set with the given map for storage.
func NewMapBackedSet(storage Map) *MapBackedSet {
	return &MapBackedSet{storage: storage}
}

// NewHashSet creates a new set backed by a HashMap.
func NewHashSet() Set {
	return NewMapBackedSet(NewHashMap())
}

// NewHashSetWithCapacity creates a new set backed by a HashMap with the given
// initial capacity.
func NewHashSetWithCapacity(capacity int) Set {
	return NewMapBackedSet(NewHashMapWithCapacity(capacity))
}

// NewLinkedHashSet creates a new set backed by a LinkedHashMap.
func NewLinkedHashSet() Set {
	return NewMapBackedSet(NewLinkedHashMap())
}

// NewLinkedHashSetWithCapacity creates a new set backed by a LinkedHashMap with
// the given initial capacity.
func NewLinkedHashSetWithCapacity(capacity int) Set {
	return NewMapBackedSet(NewLinkedHashMapWithCapacity(capacity))
}

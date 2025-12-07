package datastructure

// HashMap is a simple hash map implementation, backed by the built-in map type
// in Go.
type HashMap map[interface{}]interface{}

func (m HashMap) Contains(key interface{}) (found bool) {
	_, found = m[key]
	return
}

func (m HashMap) Put(key, value interface{}) (found bool) {
	found = m.Contains(key)
	m[key] = value

	return
}

func (m HashMap) Get(key interface{}) (value interface{}, found bool) {
	value, found = m[key]
	return
}

func (m *HashMap) GetInto(key, into interface{}) bool {
	return mapGetInto(m, key, into)
}

func (m HashMap) Remove(key interface{}) (found bool) {
	found = m.Contains(key)
	delete(m, key)

	return
}

func (m HashMap) Empty() bool {
	return m.Size() == 0
}

func (m HashMap) Size() int {
	return len(m)
}

func (m *HashMap) Clear() {
	*m = make(HashMap)
}

func (m *HashMap) Keys() []interface{} {
	return mapKeys(m)
}

func (m *HashMap) KeysInto(into interface{}) {
	mapKeysInto(m, into)
}

func (m *HashMap) Values() []interface{} {
	return mapValues(m)
}

func (m *HashMap) ValuesInto(into interface{}) {
	mapValuesInto(m, into)
}

func (m HashMap) ForEach(fn MapIterationFunc) error {
	for key, value := range m {
		if err := fn(key, value); err != nil {
			return err
		}
	}

	return nil
}

func (m *HashMap) ForEachInto(fn interface{}) error {
	return mapForEachInto(m, fn)
}

// NewHashMap creates a new hash map with the default initial capacity of this
// Go implementation.
func NewHashMap() *HashMap {
	m := make(HashMap)
	return &m
}

// NewHashMapWithCapacity creates a new hash map with the specified initial
// capacity.
func NewHashMapWithCapacity(capacity int) *HashMap {
	m := make(HashMap, capacity)
	return &m
}

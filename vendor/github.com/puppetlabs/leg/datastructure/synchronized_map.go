package datastructure

import (
	"sync"
)

// SynchronizedMap is a synchronization layer for a Map. It exposes the
// complete Map interface and passes all calls through to a given delegate,
// guarding them with a mutual exclusion lock.
//
// SynchronizedMap makes no assumptions about the underlying storage. In
// particular, it does not assume that read operations are concurrently safe.
type SynchronizedMap struct {
	storageMut sync.Mutex
	storage    Map
}

func (sm *SynchronizedMap) Empty() bool {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.Empty()
}

func (sm *SynchronizedMap) Size() int {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.Size()
}

func (sm *SynchronizedMap) Clear() {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	sm.storage.Clear()
}

func (sm *SynchronizedMap) Values() []interface{} {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.Values()
}

func (sm *SynchronizedMap) ValuesInto(into interface{}) {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	sm.storage.ValuesInto(into)
}

func (sm *SynchronizedMap) Contains(key interface{}) bool {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.Contains(key)
}

func (sm *SynchronizedMap) Put(key, value interface{}) bool {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.Put(key, value)
}

func (sm *SynchronizedMap) CompareAndPut(key, value, expected interface{}) bool {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	if v, found := sm.storage.Get(key); found && v == expected {
		sm.storage.Put(key, value)
		return true
	}

	return false
}

func (sm *SynchronizedMap) Get(key interface{}) (interface{}, bool) {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.Get(key)
}

func (sm *SynchronizedMap) GetInto(key interface{}, into interface{}) bool {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.GetInto(key, into)
}

func (sm *SynchronizedMap) Remove(key interface{}) bool {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.Remove(key)
}

func (sm *SynchronizedMap) CompareAndRemove(key, expected interface{}) bool {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	if v, found := sm.storage.Get(key); found && v == expected {
		sm.storage.Remove(key)
		return true
	}

	return false
}

func (sm *SynchronizedMap) Keys() []interface{} {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.Keys()
}

func (sm *SynchronizedMap) KeysInto(into interface{}) {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	sm.storage.KeysInto(into)
}

func (sm *SynchronizedMap) ForEach(fn MapIterationFunc) error {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.ForEach(fn)
}

func (sm *SynchronizedMap) ForEachInto(fn interface{}) error {
	sm.storageMut.Lock()
	defer sm.storageMut.Unlock()

	return sm.storage.ForEachInto(fn)
}

// NewSynchronizedMap creates a synchronization layer over a given map.
func NewSynchronizedMap(storage Map) *SynchronizedMap {
	if sm, ok := storage.(*SynchronizedMap); ok {
		return sm
	}

	return &SynchronizedMap{storage: storage}
}

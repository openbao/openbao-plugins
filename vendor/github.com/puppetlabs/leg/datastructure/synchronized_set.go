package datastructure

import (
	"sync"
)

// SynchronizedSet is a synchronization layer for a Set. It exposes the
// complete Set interface and passes all calls through to a given delegate,
// guarding them with a mutual exclusion lock.
//
// SynchronizedSet makes no assumptions about the underlying storage. In
// particular, it does not assume that read operations are concurrently safe.
type SynchronizedSet struct {
	storageMut sync.Mutex
	storage    Set
}

func (ss *SynchronizedSet) Empty() bool {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	return ss.storage.Empty()
}

func (ss *SynchronizedSet) Size() int {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	return ss.storage.Size()
}

func (ss *SynchronizedSet) Clear() {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	ss.storage.Clear()
}

func (ss *SynchronizedSet) Values() []interface{} {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	return ss.storage.Values()
}

func (ss *SynchronizedSet) ValuesInto(into interface{}) {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	ss.storage.ValuesInto(into)
}

func (ss *SynchronizedSet) Contains(elements ...interface{}) bool {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	return ss.storage.Contains(elements...)
}

func (ss *SynchronizedSet) Add(elements ...interface{}) {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	ss.storage.Add(elements...)
}

func (ss *SynchronizedSet) AddAll(other Container) {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	ss.storage.AddAll(other)
}

func (ss *SynchronizedSet) Remove(elements ...interface{}) {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	ss.storage.Remove(elements...)
}

func (ss *SynchronizedSet) RemoveAll(other Set) {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	ss.storage.RemoveAll(other)
}

func (ss *SynchronizedSet) ForEach(fn SetIterationFunc) error {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	return ss.storage.ForEach(fn)
}

func (ss *SynchronizedSet) ForEachInto(fn interface{}) error {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	return ss.storage.ForEachInto(fn)
}

func (ss *SynchronizedSet) RetainAll(other Set) {
	ss.storageMut.Lock()
	defer ss.storageMut.Unlock()

	ss.storage.RetainAll(other)
}

// NewSynchronizedSet creates a synchronization layer over a given set.
func NewSynchronizedSet(storage Set) *SynchronizedSet {
	if ss, ok := storage.(*SynchronizedSet); ok {
		return ss
	}

	return &SynchronizedSet{storage: storage}
}

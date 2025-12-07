// Portions of this file are derived from a priority queue implementation
// provided in the Go documentation.
//
// https://golang.org/pkg/container/heap/
//
// Licensed under a three-clause BSD-style license. A copy of the full license
// document is included in this distribution in the file
// `priority_queue.go.LICENSE`.

package datastructure

import (
	"container/heap"
	"reflect"
)

type priorityQueueItem struct {
	value    interface{}
	priority float64
}

type priorityQueueImpl []*priorityQueueItem

func (pqi priorityQueueImpl) Len() int {
	return len(pqi)
}

func (pqi priorityQueueImpl) Less(i, j int) bool {
	if pqi[i].priority == pqi[j].priority {
		return i < j
	}

	return pqi[i].priority > pqi[j].priority
}

func (pqi priorityQueueImpl) Swap(i, j int) {
	pqi[i], pqi[j] = pqi[j], pqi[i]
}

func (pqi *priorityQueueImpl) Push(x interface{}) {
	item := x.(*priorityQueueItem)
	*pqi = append(*pqi, item)
}

func (pqi *priorityQueueImpl) Pop() interface{} {
	old := *pqi
	n := len(old)
	item := old[n-1]
	*pqi = old[0 : n-1]
	return item
}

type PriorityQueue struct {
	impl priorityQueueImpl
}

func (pq *PriorityQueue) Empty() bool {
	return pq.Size() == 0
}

func (pq *PriorityQueue) Size() int {
	return pq.impl.Len()
}

func (pq *PriorityQueue) Clear() {
	*pq = PriorityQueue{}
	heap.Init(&pq.impl)
}

// Add inserts a new item to the priority queue with the given priority.
func (pq *PriorityQueue) Add(v interface{}, priority float64) {
	item := &priorityQueueItem{
		value:    v,
		priority: priority,
	}

	heap.Push(&pq.impl, item)
}

// Poll retrieves and removes the item with the highest priority from the queue.
//
// If no items are currently in the queue, this function returns a nil value and
// false. Otherwise, it returns the value and true.
func (pq *PriorityQueue) Poll() (interface{}, bool) {
	if pq.impl.Len() == 0 {
		return nil, false
	}

	item := heap.Pop(&pq.impl).(*priorityQueueItem)
	return item.value, true
}

// PollInto retrieves and removes the item with the highest priority from the
// queue, and stores the item value in the into parameter. The into parameter
// must be of a type assignable by the stored value. If there are no items in
// the queue, the into parameter is not modified and the function returns false.
// Otherwise, the function returns true.
//
// If the into parameter is not compatible with the stored value, this function
// will panic.
func (pq *PriorityQueue) PollInto(into interface{}) bool {
	value, found := pq.Poll()

	if found {
		target := reflect.ValueOf(into).Elem()
		target.Set(coalesceInvalidToZeroValueOf(reflect.ValueOf(value), target.Type()))
	}

	return found
}

// NewPriorityQueue creates a new priority queue backed by a heap.
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{}
	heap.Init(&pq.impl)

	return pq
}

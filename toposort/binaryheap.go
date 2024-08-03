package toposort

import (
	"golang.org/x/exp/constraints"
)

type binaryheap[T constraints.Ordered] struct {
	items []T
}

// Len returns the number of items in the PriorityQueue
func (pq binaryheap[T]) Len() int {
	return len(pq.items)
}

// Less compares the priorities of two items
func (pq binaryheap[T]) Less(i, j int) bool {
	return pq.items[i] < pq.items[j]
}

// Swap swaps the positions of two items
func (pq binaryheap[T]) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

// Push adds an item to the PriorityQueue
func (pq *binaryheap[T]) Push(x any) {
	item := x.(T)
	pq.items = append(pq.items, item)
}

// Pop removes and returns the item with the highest priority
func (pq *binaryheap[T]) Pop() any {
	old := pq.items
	n := len(old)
	item := old[n-1]
	pq.items = old[0 : n-1]
	return item
}

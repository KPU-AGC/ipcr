// internal/runutil/lru_set.go  (NEW)  — bounded dedupe utility
package runutil

import "container/list"

// LRUSet is a size-bounded set with O(1) hit/insert and FIFO eviction.
// Add returns true if the key was already present.
type LRUSet[K comparable] struct {
	cap int
	ll  *list.List
	m   map[K]*list.Element
}

type lruNode[K comparable] struct{ k K }

func NewLRUSet[K comparable](capacity int) *LRUSet[K] {
	if capacity <= 0 {
		capacity = 200_000 // sensible default for window-dedupe
	}
	return &LRUSet[K]{cap: capacity, ll: list.New(), m: make(map[K]*list.Element, capacity)}
}

// Add inserts k; returns true if it was already present.
func (s *LRUSet[K]) Add(k K) bool {
	if e, ok := s.m[k]; ok {
		s.ll.MoveToFront(e)
		return true
	}
	e := s.ll.PushFront(&lruNode[K]{k: k})
	s.m[k] = e
	if s.ll.Len() > s.cap {
		tail := s.ll.Back()
		if tail != nil {
			s.ll.Remove(tail)
			delete(s.m, tail.Value.(*lruNode[K]).k)
		}
	}
	return false
}

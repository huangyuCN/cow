package cow

import "sync/atomic"

type Store[T any] interface {
	Load() *T
	Commit(next *T)
}

type memoryStore[T any] struct {
	ptr atomic.Pointer[T]
}

func newMemoryStore[T any](root *T) *memoryStore[T] {
	store := &memoryStore[T]{}
	store.ptr.Store(root)
	return store
}

func (s *memoryStore[T]) Load() *T {
	return s.ptr.Load()
}

func (s *memoryStore[T]) Commit(next *T) {
	s.ptr.Store(next)
}

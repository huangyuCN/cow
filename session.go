package cow

import "slices"

type TxSession[T any] struct {
	store       Store[T]
	base        *T
	work        *T
	clone       func(*T) *T
	checkpoints []checkpoint[T]
	nextID      SavepointID
	dirty       DirtySet
	cloned      DirtySet
	finished    bool
}

func Mutable[T any, C any](sess *TxSession[T], pick func(root *T) *C) *C {
	return pick(sess.work)
}

func (s *TxSession[T]) ensureWritable() *T {
	if s.finished {
		panic(ErrSessionClosed)
	}
	if s.work == nil {
		work := new(T)
		*work = *s.base
		s.work = work
	}
	return s.work
}

func (s *TxSession[T]) markDirty(name string) {
	if s.dirty == nil {
		s.dirty = make(DirtySet)
	}
	s.dirty.Mark(name)
}

func (s *TxSession[T]) markCloned(name string) {
	if s.cloned == nil {
		s.cloned = make(DirtySet)
	}
	s.cloned.Mark(name)
}

func (s *TxSession[T]) Commit() error {
	if s.finished {
		return ErrSessionClosed
	}
	if s.work != nil {
		s.store.Commit(s.work)
	}
	s.finished = true
	return nil
}

func (s *TxSession[T]) Rollback() {
	s.finished = true
}

func (s *TxSession[T]) Dirty() []string {
	names := make([]string, 0, len(s.dirty))
	for name := range s.dirty {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

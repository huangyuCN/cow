package cow

type SavepointID uint64

type checkpoint[T any] struct {
	id       SavepointID
	writable bool
	root     *T
	dirty    DirtySet
}

func (s *TxSession[T]) Savepoint() (SavepointID, error) {
	if s.finished {
		return 0, ErrSessionClosed
	}
	s.nextID++
	cp := checkpoint[T]{
		id:       s.nextID,
		writable: s.work != nil,
	}
	if cp.writable {
		cp.root = s.clone(s.work)
		cp.dirty = s.dirty.Clone()
	}
	s.checkpoints = append(s.checkpoints, cp)
	return cp.id, nil
}

func (s *TxSession[T]) RollbackTo(id SavepointID) error {
	if s.finished {
		return ErrSessionClosed
	}
	n := len(s.checkpoints)
	if n == 0 || s.checkpoints[n-1].id != id {
		return ErrInvalidSavepoint
	}
	last := s.checkpoints[n-1]
	s.checkpoints = s.checkpoints[:n-1]
	if !last.writable {
		s.work = nil
		s.dirty = nil
		s.cloned = nil
		return nil
	}
	s.work = s.clone(last.root)
	s.dirty = last.dirty.Clone()
	s.cloned = nil
	return nil
}

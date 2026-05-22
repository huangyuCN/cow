package cow

func Begin[T any](store Store[T], clone func(*T) *T) (*TxSession[T], error) {
	return &TxSession[T]{
		store: store,
		base:  store.Load(),
		clone: clone,
	}, nil
}

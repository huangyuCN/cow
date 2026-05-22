package cow

func newBenchSession() *TxSession[testRoot] {
	base := newTestRoot()
	return &TxSession[testRoot]{
		base:  base,
		clone: cloneTestRoot,
	}
}

package cow

type DirtySet map[string]struct{}

func (d DirtySet) Mark(name string) {
	d[name] = struct{}{}
}

func (d DirtySet) Clone() DirtySet {
	if d == nil {
		return nil
	}
	next := make(DirtySet, len(d))
	for name := range d {
		next[name] = struct{}{}
	}
	return next
}

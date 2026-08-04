package cowgen

import (
	"go/types"
	"sort"
	"strconv"
)

// Qualifiers 维护 import path → 唯一别名，供 TypeStr 与 import 块共用。
type Qualifiers struct {
	byPath    map[string]string // import path → alias
	usedAlias map[string]string // alias → path（冲突检测）
}

// NewQualifiers 创建空的选择子表。
func NewQualifiers() *Qualifiers {
	return &Qualifiers{
		byPath:    make(map[string]string),
		usedAlias: make(map[string]string),
	}
}

// Alias 返回包的唯一选择子别名；同 path 稳定，同 Name 不同 path 则 Name+N（N≥2）。
func (q *Qualifiers) Alias(p *types.Package) string {
	if p == nil {
		return ""
	}
	path := p.Path()
	if alias, ok := q.byPath[path]; ok {
		return alias
	}
	base := p.Name()
	alias := base
	for n := 2; ; n++ {
		if prev, ok := q.usedAlias[alias]; !ok || prev == path {
			break
		}
		alias = base + strconv.Itoa(n)
	}
	q.byPath[path] = alias
	q.usedAlias[alias] = path
	return alias
}

// Imports 返回已登记的 import，按 Path 稳定排序。
func (q *Qualifiers) Imports() []struct{ Alias, Path string } {
	if q == nil || len(q.byPath) == 0 {
		return nil
	}
	paths := make([]string, 0, len(q.byPath))
	for path := range q.byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	out := make([]struct{ Alias, Path string }, len(paths))
	for i, path := range paths {
		out[i] = struct{ Alias, Path string }{Alias: q.byPath[path], Path: path}
	}
	return out
}

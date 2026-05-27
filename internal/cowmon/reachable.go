package cowmon

import (
	"fmt"
	"go/types"
)

// MonitoredSet 纳入 bare-write 检查的具名 struct（根类型 BFS 同包可达）。
type MonitoredSet struct {
	// byObj 以 *types.TypeName 为键，避免不同包同名 struct 误判。
	byObj map[*types.TypeName]struct{}
	// pkgPath 为本集合所属包路径（仅供 ContainsName 单包测试用）。
	pkgPath string
}

// ContainsName 按类型名判断（仅限 MonitoredSet 所属包内的类型名）。
func (s *MonitoredSet) ContainsName(name string) bool {
	for obj := range s.byObj {
		if obj.Name() == name && obj.Pkg().Path() == s.pkgPath {
			return true
		}
	}
	return false
}

// Contains 判断类型是否为监控 struct（含 *T），按类型对象身份匹配。
func (s *MonitoredSet) Contains(t types.Type) bool {
	named := namedStruct(t)
	if named == nil {
		return false
	}
	_, ok := s.byObj[named.Obj()]
	return ok
}

// LoadMonitored 加载包并构建监控类型集合。
func LoadMonitored(importPath string) (*MonitoredSet, error) {
	info, err := LoadPackage(importPath)
	if err != nil {
		return nil, err
	}
	reachable, err := CollectReachable(info)
	if err != nil {
		return nil, err
	}
	set := &MonitoredSet{
		byObj:   make(map[*types.TypeName]struct{}, len(reachable)),
		pkgPath: info.ImportPath,
	}
	for _, n := range reachable {
		set.byObj[n.Obj()] = struct{}{}
	}
	return set, nil
}

// CollectReachable 从根类型 BFS 收集同包 struct（与 undoproxy-gen 一致）。
func CollectReachable(pkg *PackageInfo) ([]*types.Named, error) {
	seen := make(map[string]*types.Named)
	var queue []*types.Named
	for _, r := range pkg.Roots {
		queue = append(queue, r)
	}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		name := n.Obj().Name()
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = n
		st, ok := n.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		for i := 0; i < st.NumFields(); i++ {
			refs, err := structRefsInType(st.Field(i).Type(), pkg.Pkg)
			if err != nil {
				return nil, err
			}
			for _, ref := range refs {
				if ref.Obj().Pkg().Path() != pkg.Pkg.Path() {
					return nil, fmt.Errorf("field %s.%s references external type %s", name, st.Field(i).Name(), ref.Obj().Name())
				}
				queue = append(queue, ref)
			}
		}
	}
	out := make([]*types.Named, 0, len(seen))
	for _, n := range seen {
		out = append(out, n)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i].Obj().Name() > out[j].Obj().Name() {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func structRefsInType(t types.Type, pkg *types.Package) ([]*types.Named, error) {
	var refs []*types.Named
	var walk func(types.Type) error
	walk = func(typ types.Type) error {
		switch u := typ.(type) {
		case *types.Pointer:
			return walk(u.Elem())
		case *types.Slice:
			return walk(u.Elem())
		case *types.Map:
			if err := walk(u.Key()); err != nil {
				return err
			}
			return walk(u.Elem())
		case *types.Named:
			if _, ok := u.Underlying().(*types.Struct); ok && u.Obj().Pkg() == pkg {
				refs = append(refs, u)
				return nil
			}
			return walk(u.Underlying())
		case *types.Basic:
			return nil
		default:
			return fmt.Errorf("unsupported type %s", typ.String())
		}
	}
	if err := walk(t); err != nil {
		return nil, err
	}
	return refs, nil
}

func namedStruct(t types.Type) *types.Named {
	switch u := t.(type) {
	case *types.Named:
		if _, ok := u.Underlying().(*types.Struct); ok {
			return u
		}
	case *types.Pointer:
		if n, ok := u.Elem().(*types.Named); ok {
			if _, ok := n.Underlying().(*types.Struct); ok {
				return n
			}
		}
	}
	return nil
}

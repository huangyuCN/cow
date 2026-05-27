package cowgen

import (
	"fmt"
	"go/types"
)

func classifyField(t types.Type, pkg *types.Package) (*FieldPlan, error) {
	plan := &FieldPlan{DeclaredType: TypeStr(pkg, t)}
	peeled := peelNamedContainers(t, pkg)
	return classifyType(peeled, pkg, plan, nil)
}

// peelNamedContainers 剥离同包 map/slice/指针类型别名，直到裸容器或具名 struct/basic。
func peelNamedContainers(t types.Type, pkg *types.Package) types.Type {
	for {
		n, ok := t.(*types.Named)
		if !ok || n.Obj().Pkg() != pkg {
			break
		}
		switch u := n.Underlying().(type) {
		case *types.Map, *types.Slice, *types.Pointer:
			t = u
			continue
		default:
			break
		}
		break
	}
	return t
}

func classifyType(t types.Type, pkg *types.Package, plan *FieldPlan, keys []KeyLayer) (*FieldPlan, error) {
	switch u := t.(type) {
	case *types.Pointer:
		elem := u.Elem()
		if named, ok := elem.(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Struct); ok && named.Obj().Pkg() == pkg {
				plan.Kind = KindPtrStruct
				plan.LeafType = TypeStr(pkg, t)
				plan.ElemName = named.Obj().Name()
				return plan, nil
			}
		}
		return nil, fmt.Errorf("unsupported pointer %s", t)
	case *types.Slice:
		elem := u.Elem()
		plan.SliceType = TypeStr(pkg, t)
		plan.SliceElem = TypeStr(pkg, elem)
		if len(keys) == 0 {
			if _, ok := elem.(*types.Pointer); ok {
				plan.Kind = KindSlicePtr
			} else {
				plan.Kind = KindSliceValue
			}
			if named, ok := derefNamed(elem); ok {
				plan.ElemName = named.Obj().Name()
			}
			return plan, nil
		}
		plan.Keys = keys
		if _, ok := elem.(*types.Pointer); ok {
			plan.Kind = KindMapSlicePtr
			if named, ok := derefNamed(elem); ok {
				plan.ElemName = named.Obj().Name()
			}
		} else {
			plan.Kind = KindMapSliceValue
		}
		return plan, nil
	case *types.Map:
		keyT := BasicTypeStr(pkg, u.Key())
		keys = append(keys, KeyLayer{KeyType: keyT})
		elem := u.Elem()
		if len(keys) == 1 {
			return classifyMapElem(elem, pkg, plan, keys)
		}
		if len(keys) == 2 {
			return classifyMapMapElem(elem, pkg, plan, keys)
		}
		return classifyType(elem, pkg, plan, keys)
	case *types.Basic:
		if len(keys) > 0 {
			return nil, fmt.Errorf("unexpected scalar in nested path")
		}
		plan.Kind = KindScalar
		plan.LeafType = u.Name()
		return plan, nil
	case *types.Named:
		if _, ok := u.Underlying().(*types.Basic); ok {
			if len(keys) > 0 {
				plan.Kind = KindMapScalar
				plan.LeafType = BasicTypeStr(pkg, t)
				plan.Keys = keys
				return plan, nil
			}
			plan.Kind = KindScalar
			plan.LeafType = BasicTypeStr(pkg, t)
			return plan, nil
		}
		if _, ok := u.Underlying().(*types.Struct); ok && u.Obj().Pkg() == pkg {
			if len(keys) == 0 {
				return nil, fmt.Errorf("struct value field needs map parent")
			}
			if len(keys) == 1 {
				plan.Kind = KindMapStruct
				plan.LeafType = TypeStr(pkg, t)
				plan.Keys = keys
				return plan, nil
			}
			if len(keys) == 2 {
				plan.Kind = KindMapMapStruct
				plan.LeafType = TypeStr(pkg, t)
				plan.Keys = keys
				return plan, nil
			}
		}
		return nil, fmt.Errorf("unsupported named %s", t)
	default:
		return nil, fmt.Errorf("unsupported type %s", t)
	}
}

func classifyMapElem(elem types.Type, pkg *types.Package, plan *FieldPlan, keys []KeyLayer) (*FieldPlan, error) {
	plan.Keys = keys
	if m, ok := elem.(*types.Map); ok {
		keys = append(keys, KeyLayer{KeyType: TypeStr(pkg, m.Key())})
		plan.MapValue = TypeStr(pkg, elem)
		return classifyMapMapElem(m.Elem(), pkg, plan, keys)
	}
	switch e := elem.(type) {
	case *types.Basic:
		plan.Kind = KindMapScalar
		plan.LeafType = e.Name()
		return plan, nil
	case *types.Pointer:
		if named, ok := e.Elem().(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Struct); ok && named.Obj().Pkg() == pkg {
				plan.Kind = KindMapPtrStruct
				plan.LeafType = TypeStr(pkg, elem)
				plan.ElemName = Singular(plan.FieldName)
				if plan.ElemName == plan.FieldName {
					plan.ElemName = named.Obj().Name()
				}
				return plan, nil
			}
		}
	case *types.Slice:
		plan.SliceType = TypeStr(pkg, e)
		plan.SliceElem = TypeStr(pkg, e.Elem())
		plan.Keys = keys
		if _, ok := e.Elem().(*types.Pointer); ok {
			plan.Kind = KindMapSlicePtr
			if named, ok := derefNamed(e.Elem()); ok {
				plan.ElemName = named.Obj().Name()
			}
		} else {
			plan.Kind = KindMapSliceValue
		}
		return plan, nil
	case *types.Named:
		if _, ok := e.Underlying().(*types.Struct); ok && e.Obj().Pkg() == pkg {
			plan.Kind = KindMapStruct
			plan.LeafType = TypeStr(pkg, elem)
			return plan, nil
		}
	}
	return nil, fmt.Errorf("unsupported map elem %s", elem)
}

func classifyMapMapElem(elem types.Type, pkg *types.Package, plan *FieldPlan, keys []KeyLayer) (*FieldPlan, error) {
	plan.Keys = keys
	switch e := elem.(type) {
	case *types.Basic:
		plan.Kind = KindMapMapScalar
		plan.LeafType = TypeStr(pkg, elem)
		return plan, nil
	case *types.Pointer:
		if named, ok := e.Elem().(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Struct); ok && named.Obj().Pkg() == pkg {
				plan.Kind = KindMapMapPtrStruct
				plan.LeafType = TypeStr(pkg, elem)
				plan.ElemName = named.Obj().Name()
				return plan, nil
			}
		}
	case *types.Slice:
		plan.SliceType = TypeStr(pkg, e)
		plan.SliceElem = TypeStr(pkg, e.Elem())
		plan.MapValue = "map[" + keys[1].KeyType + "]" + plan.SliceType
		if _, ok := e.Elem().(*types.Pointer); ok {
			plan.Kind = KindMapMapSlicePtr
		} else {
			plan.Kind = KindMapMapSliceValue
		}
		if named, ok := derefNamed(e.Elem()); ok {
			plan.ElemName = named.Obj().Name()
		}
		return plan, nil
	case *types.Named:
		if _, ok := e.Underlying().(*types.Struct); ok && e.Obj().Pkg() == pkg {
			plan.Kind = KindMapMapStruct
			plan.LeafType = TypeStr(pkg, elem)
			return plan, nil
		}
	}
	return nil, fmt.Errorf("unsupported map map elem %s", elem)
}

func derefNamed(t types.Type) (*types.Named, bool) {
	if p, ok := t.(*types.Pointer); ok {
		if n, ok := p.Elem().(*types.Named); ok {
			return n, true
		}
	}
	if n, ok := t.(*types.Named); ok {
		return n, true
	}
	return nil, false
}

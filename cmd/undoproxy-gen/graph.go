package main

import (
	"fmt"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/huangyuCN/cow/internal/cowmon"
)

// FieldKind 字段分类，驱动代码生成。
type FieldKind int

const (
	KindScalar FieldKind = iota
	KindPtrStruct
	KindMapScalar
	KindMapStruct
	KindMapPtrStruct
	KindSliceValue
	KindSlicePtr
	KindMapSliceValue
	KindMapSlicePtr
	KindMapMapScalar
	KindMapMapStruct
	KindMapMapPtrStruct
	KindMapMapSliceValue
	KindMapMapSlicePtr
)

// KeyLayer map/slice 路径的一层。
type KeyLayer struct {
	KeyType string // 打印后的 key 类型，如 int32
}

// FieldPlan 单个字段对应的生成方法。
type FieldPlan struct {
	FieldName string
	Kind      FieldKind
	Keys      []KeyLayer // 外层→内层 map key
	LeafType   string // 元素/值类型打印
	MapValue   string // 内层 map 值类型打印（MapMap 时）
	SliceType  string // 完整 slice 类型，如 []*Item
	SliceElem  string // slice 元素类型
	ElemName  string     // Get 用 Singular
}

// StructPlan 一个 struct 的全部生成内容。
type StructPlan struct {
	Name   string
	Struct *types.Struct
	Plans  []FieldPlan
	Clone  bool
}

// Graph 生成计划。
type Graph struct {
	Structs []*StructPlan
}

func buildGraph(pkg *PackageInfo) (*Graph, error) {
	reachable, err := cowmon.CollectReachable(pkg)
	if err != nil {
		return nil, err
	}
	g := &Graph{}
	for _, named := range reachable {
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		sp := &StructPlan{Name: named.Obj().Name(), Clone: true, Struct: st}
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			if !f.Exported() {
				continue
			}
			plan, err := classifyField(f.Type(), pkg.Pkg)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", sp.Name, f.Name(), err)
			}
			if plan != nil {
				plan.FieldName = f.Name()
				sp.Plans = append(sp.Plans, *plan)
			}
		}
		g.Structs = append(g.Structs, sp)
	}
	return g, nil
}

func typeStr(pkg *types.Package, t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string {
		if p == nil || p == pkg {
			return ""
		}
		return p.Name()
	})
}

func classifyField(t types.Type, pkg *types.Package) (*FieldPlan, error) {
	plan := &FieldPlan{}
	return classifyType(t, pkg, plan, nil)
}

func classifyType(t types.Type, pkg *types.Package, plan *FieldPlan, keys []KeyLayer) (*FieldPlan, error) {
	switch u := t.(type) {
	case *types.Pointer:
		elem := u.Elem()
		if named, ok := elem.(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Struct); ok && named.Obj().Pkg() == pkg {
				plan.Kind = KindPtrStruct
				plan.LeafType = typeStr(pkg, t)
				plan.ElemName = named.Obj().Name()
				return plan, nil
			}
		}
		return nil, fmt.Errorf("unsupported pointer %s", t)
	case *types.Slice:
		elem := u.Elem()
		plan.SliceType = typeStr(pkg, t)
		plan.SliceElem = typeStr(pkg, elem)
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
		keyT := typeStr(pkg, u.Key())
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
		plan.LeafType = t.String()
		return plan, nil
	case *types.Named:
		if _, ok := u.Underlying().(*types.Basic); ok {
			if len(keys) > 0 {
				plan.Kind = KindMapScalar
				plan.LeafType = typeStr(pkg, t)
				plan.Keys = keys
				return plan, nil
			}
			plan.Kind = KindScalar
			plan.LeafType = typeStr(pkg, t)
			return plan, nil
		}
		if _, ok := u.Underlying().(*types.Struct); ok && u.Obj().Pkg() == pkg {
			if len(keys) == 0 {
				return nil, fmt.Errorf("struct value field needs map parent")
			}
			if len(keys) == 1 {
				plan.Kind = KindMapStruct
				plan.LeafType = typeStr(pkg, t)
				plan.Keys = keys
				return plan, nil
			}
			if len(keys) == 2 {
				plan.Kind = KindMapMapStruct
				plan.LeafType = typeStr(pkg, t)
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
		keys = append(keys, KeyLayer{KeyType: typeStr(pkg, m.Key())})
		plan.MapValue = typeStr(pkg, elem)
		return classifyMapMapElem(m.Elem(), pkg, plan, keys)
	}
	switch e := elem.(type) {
	case *types.Basic:
		plan.Kind = KindMapScalar
		plan.LeafType = typeStr(pkg, elem)
		return plan, nil
	case *types.Pointer:
		if named, ok := e.Elem().(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Struct); ok && named.Obj().Pkg() == pkg {
				plan.Kind = KindMapPtrStruct
				plan.LeafType = typeStr(pkg, elem)
				plan.ElemName = singular(plan.FieldName)
				if plan.ElemName == plan.FieldName {
					plan.ElemName = named.Obj().Name()
				}
				return plan, nil
			}
		}
	case *types.Slice:
		plan.SliceType = typeStr(pkg, e)
		plan.SliceElem = typeStr(pkg, e.Elem())
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
			plan.LeafType = typeStr(pkg, elem)
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
		plan.LeafType = typeStr(pkg, elem)
		return plan, nil
	case *types.Pointer:
		if named, ok := e.Elem().(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Struct); ok && named.Obj().Pkg() == pkg {
				plan.Kind = KindMapMapPtrStruct
				plan.LeafType = typeStr(pkg, elem)
				plan.ElemName = named.Obj().Name()
				return plan, nil
			}
		}
	case *types.Slice:
		plan.SliceType = typeStr(pkg, e)
		plan.SliceElem = typeStr(pkg, e.Elem())
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
			plan.LeafType = typeStr(pkg, elem)
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

func recvIdent(structName string) string {
	if structName == "" {
		return "x"
	}
	r, _ := utf8.DecodeRuneInString(structName)
	return string(unicode.ToLower(r))
}

func keyParams(keys []KeyLayer) string {
	var ps []string
	for i, k := range keys {
		ps = append(ps, fmt.Sprintf("k%d %s", i+1, k.KeyType))
	}
	return strings.Join(ps, ", ")
}

func keyArgs(keys []KeyLayer) string {
	var as []string
	for i := range keys {
		as = append(as, fmt.Sprintf("k%d", i+1))
	}
	return strings.Join(as, ", ")
}

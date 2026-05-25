package main

import (
	"go/ast"
	"go/types"

	"github.com/huangyuCN/cow/internal/cowmon"
)

type writePath struct {
	Root       ast.Expr
	RootStruct string
	Nav        []navStep
	LeafField  string
	MapKeys    []ast.Expr
	SliceIndex ast.Expr
	Kind       writeKind
}

type navStep struct {
	Field string
	MapKey ast.Expr
}

type writeKind int

const (
	writePut writeKind = iota
	writeAppend
	writeSetAt
	writeMapPut
	writeMapMapPut
)

type pathPart struct {
	tag   string // field, mapKey, sliceKey
	field string
	key   ast.Expr
}

func collectParts(expr ast.Expr, info *types.Info) (ast.Expr, []pathPart) {
	var parts []pathPart
	for {
		switch e := expr.(type) {
		case *ast.SelectorExpr:
			parts = append([]pathPart{{tag: "field", field: e.Sel.Name}}, parts...)
			expr = e.X
		case *ast.IndexExpr:
			tag := "sliceKey"
			if t := info.TypeOf(e.X); t != nil {
				if _, ok := t.Underlying().(*types.Map); ok {
					tag = "mapKey"
				}
			}
			parts = append([]pathPart{{tag: tag, key: e.Index}}, parts...)
			expr = e.X
		default:
			return expr, parts
		}
	}
}

func parseWritePath(lhs ast.Expr, info *types.Info, mon *cowmon.MonitoredSet) (*writePath, bool) {
	root, parts := collectParts(lhs, info)
	named := namedStruct(root, info)
	if named == nil || !mon.Contains(named) {
		return nil, false
	}
	if len(parts) == 0 {
		return nil, false
	}
	wp := &writePath{Root: root, RootStruct: named.Obj().Name()}
	last := parts[len(parts)-1]
	prefix := parts[:len(parts)-1]
	switch last.tag {
	case "field":
		wp.LeafField = last.field
	case "sliceKey":
		wp.SliceIndex = last.key
		if len(prefix) == 0 || prefix[len(prefix)-1].tag != "field" {
			return nil, false
		}
		wp.LeafField = prefix[len(prefix)-1].field
		prefix = prefix[:len(prefix)-1]
	case "mapKey":
		wp.MapKeys = append([]ast.Expr{last.key}, wp.MapKeys...)
	default:
		return nil, false
	}
	for i := 0; i < len(prefix); i++ {
		switch prefix[i].tag {
		case "mapKey":
			wp.MapKeys = append(wp.MapKeys, prefix[i].key)
		case "sliceKey":
			wp.SliceIndex = prefix[i].key
		case "field":
			if wp.LeafField == "" {
				wp.LeafField = prefix[i].field
			} else {
				var mk ast.Expr
				if i+1 < len(prefix) && prefix[i+1].tag == "mapKey" {
					mk = prefix[i+1].key
					i++
				}
				wp.Nav = append(wp.Nav, navStep{Field: prefix[i].field, MapKey: mk})
			}
		}
	}
	if wp.LeafField == "" {
		return nil, false
	}
	// 若尚未建 Nav，补一条仅 field 的步（无 mapKey 的标量字段）
	if len(wp.Nav) == 0 && len(wp.MapKeys) == 0 && wp.SliceIndex == nil {
		// 叶字段直接在 root 上
	}
	if len(wp.MapKeys) >= 2 {
		// parts 自叶向根收集，键顺序需反转为 k1,k2
		for i, j := 0, len(wp.MapKeys)-1; i < j; i, j = i+1, j-1 {
			wp.MapKeys[i], wp.MapKeys[j] = wp.MapKeys[j], wp.MapKeys[i]
		}
	}
	if wp.SliceIndex != nil {
		wp.Kind = writeSetAt
	} else if len(wp.MapKeys) >= 2 {
		wp.Kind = writeMapMapPut
	} else if len(wp.MapKeys) == 1 {
		wp.Kind = writeMapPut
	} else {
		wp.Kind = writePut
	}
	return wp, true
}

func namedStruct(expr ast.Expr, info *types.Info) *types.Named {
	tv := info.Types[expr]
	if tv.Type == nil {
		return nil
	}
	switch t := tv.Type.(type) {
	case *types.Pointer:
		if n, ok := t.Elem().(*types.Named); ok {
			if _, ok := n.Underlying().(*types.Struct); ok {
				return n
			}
		}
	case *types.Named:
		if _, ok := t.Underlying().(*types.Struct); ok {
			return t
		}
	}
	return nil
}

// receiverStruct 沿 Nav 走完后的 struct 类型名（用于查 Leaf Put）。
func receiverStruct(root string, nav []navStep, cat lookupCatalog) string {
	cur := root
	for _, step := range nav {
		fm, ok := cat.Lookup(cur, step.Field)
		if !ok {
			return ""
		}
		if fm.TargetStruct != "" {
			cur = fm.TargetStruct
		}
	}
	return cur
}

func structFromGetMethod(method string) string {
	if len(method) > 9 && method[len(method)-9:] == "ForWrite" {
		s := method[3 : len(method)-8]
		if len(s) > 0 {
			return s
		}
	}
	if len(method) > 11 && method[len(method)-11:] == "AtForWrite" {
		return method[3 : len(method)-11]
	}
	return ""
}

type lookupCatalog interface {
	Lookup(structName, fieldName string) (fieldMethods, bool)
}

type fieldMethods struct {
	Put, Append, SetAt, GetForWrite, GetAtForWrite string
	MapPutKeyCount                                 int
	TargetStruct                                   string
}

package cowproxy

import (
	"github.com/huangyuCN/cow/internal/cowgen"
	"github.com/huangyuCN/cow/internal/cowmon"
)

// FieldMethods 某 struct 某字段的代理方法名（与 undoproxy-gen 一致）。
type FieldMethods struct {
	Put            string
	Set            string
	Remove         string
	Append         string
	SetAt          string
	RemoveAt       string
	Truncate       string
	GetForWrite    string
	GetAtForWrite  string
	MapForWrite    string
	MapPutKeyCount int
	TargetStruct   string // Get*ForWrite 指向的子 struct 名
}

// RewriteCatalog 改写用的 struct→field→方法表。
type RewriteCatalog struct {
	ByStruct map[string]map[string]FieldMethods
}

// NewCatalog 从 cow 包类型图构建改写目录。
func NewCatalog(cowImport string) (*RewriteCatalog, error) {
	pkg, err := cowmon.LoadPackage(cowImport)
	if err != nil {
		return nil, err
	}
	g, err := cowgen.BuildGraph(pkg)
	if err != nil {
		return nil, err
	}
	cat := &RewriteCatalog{ByStruct: make(map[string]map[string]FieldMethods)}
	for _, sp := range g.Structs {
		fields := make(map[string]FieldMethods)
		for _, plan := range sp.Plans {
			fields[plan.FieldName] = methodsFromPlan(plan)
		}
		cat.ByStruct[sp.Name] = fields
	}
	return cat, nil
}

func methodsFromPlan(plan cowgen.FieldPlan) FieldMethods {
	fm := FieldMethods{Put: cowgen.PutFieldName(plan.FieldName)}
	switch plan.Kind {
	case cowgen.KindScalar:
		fm.MapPutKeyCount = 0
	case cowgen.KindPtrStruct:
		fm.GetForWrite = cowgen.PtrGetForWriteName(plan.FieldName)
		fm.Set = cowgen.PtrSetName(plan.FieldName)
		fm.TargetStruct = plan.ElemName
	case cowgen.KindMapScalar, cowgen.KindMapStruct, cowgen.KindMapPtrStruct:
		fm.MapPutKeyCount = len(plan.Keys)
		fm.Remove = cowgen.MapRemoveName(plan.FieldName)
		if plan.Kind == cowgen.KindMapPtrStruct {
			fm.GetForWrite = cowgen.MapKeyGetForWriteName(plan.ElemName)
			fm.TargetStruct = plan.ElemName
		}
	case cowgen.KindSliceValue, cowgen.KindSlicePtr:
		s := cowgen.SliceMethodNames(plan.FieldName)
		fm.Append, fm.SetAt, fm.RemoveAt, fm.Truncate = s.Append, s.SetAt, s.RemoveAt, s.Truncate
	case cowgen.KindMapSliceValue, cowgen.KindMapSlicePtr:
		s := cowgen.SliceMethodNames(plan.FieldName)
		fm.Append, fm.SetAt, fm.RemoveAt, fm.Truncate = s.Append, s.SetAt, s.RemoveAt, s.Truncate
		fm.MapPutKeyCount = 1
		if plan.ElemName != "" {
			fm.GetAtForWrite = cowgen.ElemAtForWriteName(plan.ElemName)
			fm.TargetStruct = plan.ElemName
		}
	case cowgen.KindMapMapScalar, cowgen.KindMapMapStruct, cowgen.KindMapMapPtrStruct,
		cowgen.KindMapMapSliceValue, cowgen.KindMapMapSlicePtr:
		fm.MapPutKeyCount = 2
		fm.Remove = cowgen.MapRemoveName(plan.FieldName)
		fm.MapForWrite = cowgen.MapForWriteName(plan.FieldName)
		if plan.Kind == cowgen.KindMapMapPtrStruct {
			// 双层 map 的 *Struct 值用 Put 双层键，无单独 Get
		}
	}
	return fm
}

// Lookup 查询 struct 字段的方法表。
func (c *RewriteCatalog) Lookup(structName, fieldName string) (FieldMethods, bool) {
	m, ok := c.ByStruct[structName]
	if !ok {
		return FieldMethods{}, false
	}
	fm, ok := m[fieldName]
	return fm, ok
}

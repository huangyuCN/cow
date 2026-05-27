package cowgen

import "go/types"

// FieldKind 字段分类，驱动代码生成与改写目录。
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
	KeyType string
}

// FieldPlan 单个字段对应的生成/改写方法计划。
type FieldPlan struct {
	FieldName    string
	Kind         FieldKind
	Keys         []KeyLayer
	LeafType     string
	DeclaredType string // 字段声明类型名（含 map/slice 别名，如 Equips、ItemList）
	MapValue     string
	SliceType    string
	SliceElem    string
	ElemName     string
}

// StructPlan 一个 struct 的全部字段计划。
type StructPlan struct {
	Name   string
	Struct *types.Struct
	Plans  []FieldPlan
	Clone  bool
}

// Graph 类型图生成/改写计划。
type Graph struct {
	Structs []*StructPlan
}

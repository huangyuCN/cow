package main

import (
	"strings"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func mapTypeString(plan cowgen.FieldPlan) string {
	if alias := containerAliasType(plan); alias != "" {
		return alias
	}
	if plan.SliceType != "" && len(plan.Keys) >= 1 {
		return "map[" + plan.Keys[0].KeyType + "]" + plan.SliceType
	}
	if len(plan.Keys) == 2 && plan.MapValue != "" {
		return "map[" + plan.Keys[0].KeyType + "]" + plan.MapValue
	}
	if len(plan.Keys) == 1 && plan.LeafType != "" {
		return "map[" + plan.Keys[0].KeyType + "]" + plan.LeafType
	}
	return ""
}

func valueTypeForMap(plan cowgen.FieldPlan) string {
	return plan.LeafType
}

func mapTypeFromPlan(plan cowgen.FieldPlan) string {
	if alias := containerAliasType(plan); alias != "" {
		return alias
	}
	if len(plan.Keys) == 1 {
		return "map[" + plan.Keys[0].KeyType + "]" + plan.LeafType
	}
	return "map[" + plan.Keys[0].KeyType + "]" + plan.MapValue
}

// containerAliasType 返回 map/slice 类型别名名（如 Equips）；字面 map[...] 返回空。
func containerAliasType(plan cowgen.FieldPlan) string {
	dt := plan.DeclaredType
	if dt == "" || strings.HasPrefix(dt, "map[") || strings.HasPrefix(dt, "[]") {
		return ""
	}
	switch plan.Kind {
	case cowgen.KindMapScalar, cowgen.KindMapStruct, cowgen.KindMapPtrStruct,
		cowgen.KindMapSliceValue, cowgen.KindMapSlicePtr,
		cowgen.KindMapMapScalar, cowgen.KindMapMapStruct, cowgen.KindMapMapPtrStruct,
		cowgen.KindMapMapSliceValue, cowgen.KindMapMapSlicePtr,
		cowgen.KindSliceValue, cowgen.KindSlicePtr:
		return dt
	default:
		return ""
	}
}

func sliceDeclaredType(plan cowgen.FieldPlan) string {
	if alias := containerAliasType(plan); alias != "" && (plan.Kind == cowgen.KindSliceValue || plan.Kind == cowgen.KindSlicePtr) {
		return alias
	}
	return plan.SliceType
}

func innerValueType(plan cowgen.FieldPlan) string {
	return plan.LeafType
}

func recvLower(structName string) string {
	if structName == "" {
		return "x"
	}
	return strings.ToLower(structName[:1]) + structName[1:]
}

func ptrSlotName(leafType string) string {
	return recvLower(strings.TrimPrefix(leafType, "*"))
}

// indexParamName 返回 slice 下标形参名，避免与 receiver（如 Item→i）同名遮蔽。
func indexParamName(recv string) string {
	if recv == "i" {
		return "idx"
	}
	return "i"
}

// truncateLenParamName 返回 Truncate 目标长度形参名，避免与 receiver（如 NodeData→n）同名遮蔽。
func truncateLenParamName(recv string) string {
	if recv == "n" {
		return "newLen"
	}
	return "n"
}

func mapKeyField(keyType string) string {
	switch keyType {
	case "int32":
		return "keyI32"
	case "int64":
		return "keyI64"
	case "uint32":
		return "keyU32"
	case "uint64":
		return "keyU64"
	default:
		return "keyString"
	}
}

package main

import (
	"strings"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func mapTypeString(plan cowgen.FieldPlan) string {
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
	if len(plan.Keys) == 1 {
		return "map[" + plan.Keys[0].KeyType + "]" + plan.LeafType
	}
	return "map[" + plan.Keys[0].KeyType + "]" + plan.MapValue
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

func oldStoreField(goType string) string {
	switch goType {
	case "int32":
		return "oldI32"
	case "int64":
		return "oldI64"
	case "uint64":
		return "oldU64"
	case "int", "uint", "uintptr":
		return "oldInt"
	case "string":
		return "oldString"
	default:
		return "oldI64"
	}
}

// leafStoreField 返回 undoOp 中存放叶子旧值的字段名（标量或 *T 指针槽）。
func leafStoreField(goType string) string {
	if strings.HasPrefix(goType, "*") {
		return ptrSlotName(goType)
	}
	return oldStoreField(goType)
}

func ptrSlotName(leafType string) string {
	return recvLower(strings.TrimPrefix(leafType, "*"))
}

func mapKeyField(keyType string) string {
	switch keyType {
	case "int32":
		return "keyI32"
	case "int64":
		return "keyI64"
	case "uint64":
		return "keyU64"
	default:
		return "keyString"
	}
}

package main

import "fmt"

type writeKind int

const (
	writeScalar writeKind = iota
	writeSliceAppend
	writeMapIndex
)

// suggestProxy 根据字段名与写操作类型给出代理方法提示。
func suggestProxy(typeName, fieldName string, kind writeKind) string {
	switch kind {
	case writeSliceAppend:
		return fmt.Sprintf("Append%s(ctx, …)", fieldName)
	case writeMapIndex:
		return fmt.Sprintf("Put%s(ctx, key, …)", fieldName)
	default:
		return fmt.Sprintf("Put%s(ctx, …)", fieldName)
	}
}

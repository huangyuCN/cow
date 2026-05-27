package main

import (
	"bytes"
	"fmt"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func emitStructuredPtrSet(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := plan.FieldName
	typ := plan.LeafType
	slot := ptrSlotName(typ)
	recv := recvLower(structName)
	kind := ub.kind(structName, field, "PtrSet",
		fmt.Sprintf("op.%s.%s = op.%s", recv, field, slot))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, val %s) {\n", r, structName, cowgen.PtrSetName(field), typ)
	fmt.Fprintf(b, "\told := %s\n", acc)
	// 整槽替换：直接接管调用方指针，不 Clone（与 Get*ForWrite 就地 COW 区分）
	fmt.Fprintf(b, "\t%s = val\n", acc)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: old})\n", kind, ub.recvArg(structName, r), slot)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapRemove(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := plan.FieldName
	kp := cowgen.KeyParams(plan.Keys)
	ka := cowgen.KeyArgs(plan.Keys)
	recv := recvLower(structName)
	keyField := mapKeyField(plan.Keys[0].KeyType)
	oldF := ub.leafStoreField(plan.LeafType)
	kind := ub.kind(structName, field, "MapKeyRemove",
		fmt.Sprintf("op.%s.%s[op.%s] = op.%s", recv, field, keyField, oldF))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s) {\n", r, structName, cowgen.MapRemoveName(field), kp)
	fmt.Fprintf(b, "\tif %s == nil {\n\t\treturn\n\t}\n", acc)
	fmt.Fprintf(b, "\told, existed := %s[%s]\n", acc, ka)
	fmt.Fprintf(b, "\tif !existed {\n\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\tdelete(%s, %s)\n", acc, ka)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: old, had: true})\n",
		kind, ub.recvArg(structName, r), keyField, ka, oldF)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapMapRemove(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := plan.FieldName
	kp := cowgen.KeyParams(plan.Keys)
	recv := recvLower(structName)
	oldF := ub.leafStoreField(plan.LeafType)
	kindInner := ub.kind(structName, field, "MapMapInnerKeyRemove",
		fmt.Sprintf(`inner := op.%s.%s[op.keyI32]
if op.had { inner[op.keyString] = op.%s } else { delete(inner, op.keyString) }`, recv, field, oldF))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s) {\n", r, structName, cowgen.MapRemoveName(field), kp)
	fmt.Fprintf(b, "\tif %s == nil {\n\t\treturn\n\t}\n", acc)
	fmt.Fprintf(b, "\tinner, ok := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif !ok || inner == nil {\n\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\told, existed := inner[k2]\n")
	fmt.Fprintf(b, "\tif !existed {\n\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\tdelete(inner, k2)\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, keyI32: k1, keyString: k2, %s: old, had: true})\n",
		kindInner, ub.recvArg(structName, r), oldF)
	fmt.Fprintf(b, "}\n\n")
}

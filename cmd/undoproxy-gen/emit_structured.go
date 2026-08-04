package main

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func emitStructuredMethods(b *bytes.Buffer, ub *undoBuilder, structName string, plan cowgen.FieldPlan) {
	r := cowgen.RecvIdent(structName)
	acc := r + "." + plan.FieldName
	switch plan.Kind {
	case cowgen.KindScalar:
		emitStructuredScalarPut(b, ub, structName, r, acc, plan)
	case cowgen.KindPtrStruct:
		emitStructuredPtrGetForWrite(b, ub, structName, r, acc, plan.FieldName, plan.LeafType)
		emitStructuredPtrSet(b, ub, structName, r, acc, plan)
	case cowgen.KindMapScalar, cowgen.KindMapStruct:
		emitStructuredMapPut(b, ub, structName, r, acc, plan, false)
		emitStructuredMapRemove(b, ub, structName, r, acc, plan)
	case cowgen.KindMapPtrStruct:
		emitStructuredMapPtrGet(b, ub, structName, r, acc, plan)
		emitStructuredMapPut(b, ub, structName, r, acc, plan, true)
		emitStructuredMapRemove(b, ub, structName, r, acc, plan)
	case cowgen.KindSliceValue, cowgen.KindSlicePtr:
		emitStructuredSliceOps(b, ub, structName, r, acc, plan, "")
	case cowgen.KindMapSliceValue, cowgen.KindMapSlicePtr:
		emitStructuredMapSliceOps(b, ub, structName, r, acc, plan)
	case cowgen.KindMapMapScalar, cowgen.KindMapMapStruct:
		emitStructuredMapMapPut(b, ub, structName, r, acc, plan, false)
		emitStructuredMapMapGetForWrite(b, ub, structName, r, acc, plan)
		emitStructuredMapMapRemove(b, ub, structName, r, acc, plan)
	case cowgen.KindMapMapPtrStruct:
		emitStructuredMapMapPtrGet(b, ub, structName, r, acc, plan)
		emitStructuredMapMapPut(b, ub, structName, r, acc, plan, true)
		emitStructuredMapMapGetForWrite(b, ub, structName, r, acc, plan)
		emitStructuredMapMapRemove(b, ub, structName, r, acc, plan)
	case cowgen.KindMapMapSliceValue, cowgen.KindMapMapSlicePtr:
		emitStructuredMapMapSliceOps(b, ub, structName, r, acc, plan)
	}
}

func fieldFromAcc(r, acc string) string {
	return strings.TrimPrefix(acc, r+".")
}

func emitStructuredScalarPut(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := fieldFromAcc(r, acc)
	oldF := ub.scalarOldField(plan.LeafType)
	recv := recvLower(structName)
	kind := ub.kind(structName, field, "ScalarSet",
		fmt.Sprintf("op.%s.%s = op.%s", recv, field, oldF))
	fmt.Fprintf(b, "func (%s *%s) Put%s(ctx *TxContext, val %s) {\n", r, structName, plan.FieldName, plan.LeafType)
	fmt.Fprintf(b, "\told := %s\n", acc)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: old})\n", kind, ub.recvArg(structName, r), oldF)
	fmt.Fprintf(b, "\t%s = val\n}\n\n", acc)
}

func emitStructuredPtrGetForWrite(b *bytes.Buffer, ub *undoBuilder, structName, r, acc, field, typ string) {
	slot := ptrSlotName(typ)
	recv := recvLower(structName)
	kind := ub.kind(structName, field, "PtrReplace",
		fmt.Sprintf("op.%s.%s = op.%s", recv, field, slot))
	fmt.Fprintf(b, "func (%s *%s) Get%sForWrite(ctx *TxContext) %s {\n", r, structName, field, typ)
	fmt.Fprintf(b, "\told := %s\n", acc)
	fmt.Fprintf(b, "\tif old == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tdirty := old.CloneForWrite()\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: old})\n", kind, ub.recvArg(structName, r), slot)
	fmt.Fprintf(b, "\t%s = dirty\n", acc)
	fmt.Fprintf(b, "\treturn dirty\n}\n\n")
}

func emitStructuredMapEnsure(b *bytes.Buffer, ub *undoBuilder, structName, r, acc, mapType string) {
	field := fieldFromAcc(r, acc)
	recv := recvLower(structName)
	kind := ub.kind(structName, field, "MapEnsureNil", fmt.Sprintf("op.%s.%s = nil", recv, field))
	fmt.Fprintf(b, "\tif %s == nil {\n", acc)
	fmt.Fprintf(b, "\t\tctx.push(undoOp{kind: %s, %s})\n", kind, ub.recvArg(structName, r))
	fmt.Fprintf(b, "\t\t%s = make(%s)\n", acc, mapType)
	fmt.Fprintf(b, "\t}\n")
}

func emitStructuredMapPut(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, ptrClone bool) {
	field := plan.FieldName
	kp := cowgen.KeyParams(plan.Keys)
	ka := cowgen.KeyArgs(plan.Keys)
	valType := plan.LeafType
	if plan.Kind == cowgen.KindMapStruct {
		valType = strings.TrimPrefix(valType, "*")
	}
	recv := recvLower(structName)
	keyField := ub.keySlot(plan.Keys[0].KeyType)
	oldF := ub.leafStoreField(plan.LeafType)
	kind := ub.kind(structName, field, "MapKeySet",
		fmt.Sprintf("if op.had { op.%s.%s[op.%s] = op.%s } else { delete(op.%s.%s, op.%s) }",
			recv, field, keyField, oldF, recv, field, keyField))
	fmt.Fprintf(b, "func (%s *%s) Put%s(ctx *TxContext, %s, val %s) {\n", r, structName, plan.FieldName, kp, valType)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeFromPlan(plan))
	fmt.Fprintf(b, "\told, existed := %s[%s]\n", acc, ka)
	if ptrClone {
		fmt.Fprintf(b, "\tif val != nil {\n\t\tval = val.CloneForWrite()\n\t}\n")
	}
	fmt.Fprintf(b, "\t%s[%s] = val\n", acc, ka)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: old, had: existed})\n",
		kind, ub.recvArg(structName, r), keyField, ka, oldF)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapPtrGet(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	ret := plan.LeafType
	slot := ptrSlotName(plan.LeafType)
	recv := recvLower(structName)
	keyField := ub.keySlot(plan.Keys[0].KeyType)
	kind := ub.kind(structName, field, "MapPtrReplace",
		fmt.Sprintf("op.%s.%s[op.%s] = op.%s", recv, field, keyField, slot))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s) %s {\n",
		r, structName, cowgen.MapKeyGetForWriteName(plan.FieldName), cowgen.KeyParams(plan.Keys), ret)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeFromPlan(plan))
	fmt.Fprintf(b, "\told, ok := %s[%s]\n", acc, ka)
	fmt.Fprintf(b, "\tif !ok || old == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tdirty := old.CloneForWrite()\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: old})\n",
		kind, ub.recvArg(structName, r), keyField, ka, slot)
	fmt.Fprintf(b, "\t%s[%s] = dirty\n", acc, ka)
	fmt.Fprintf(b, "\treturn dirty\n}\n\n")
}

func emitStructuredSliceOps(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, keyPrefix string) {
	names := cowgen.SliceMethodNames(plan.FieldName)
	elem := plan.SliceElem
	if keyPrefix != "" {
		emitStructuredMapSliceAppend(b, ub, structName, r, acc, plan, names, elem, keyPrefix)
		emitStructuredMapSliceSet(b, ub, structName, r, acc, plan, names, elem, keyPrefix)
		emitStructuredMapSliceRemove(b, ub, structName, r, acc, plan, names, elem, keyPrefix)
		emitStructuredMapSliceTruncate(b, ub, structName, r, acc, plan, names, keyPrefix)
		if plan.Kind == cowgen.KindMapSlicePtr || plan.Kind == cowgen.KindMapMapSlicePtr {
			emitStructuredMapSliceElemGet(b, ub, structName, r, acc, plan, keyPrefix)
		}
		return
	}
	field := plan.FieldName
	recv := recvLower(structName)
	idx := indexParamName(r)
	truncLen := truncateLenParamName(r)
	kindTrunc := ub.kind(structName, field, "SliceTruncate",
		fmt.Sprintf("op.%s.%s = op.%s.%s[:op.oldInt]", recv, field, recv, field))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, elem %s) {\n", r, structName, names.Append, elem)
	fmt.Fprintf(b, "\toldLen := len(%s)\n", acc)
	fmt.Fprintf(b, "\t%s = append(%s, elem)\n", acc, acc)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, oldInt: oldLen})\n", kindTrunc, ub.recvArg(structName, r))
	fmt.Fprintf(b, "}\n\n")

	oldF := ub.leafStoreField(elem)
	kindSet := ub.kind(structName, field, "SliceSetAt",
		fmt.Sprintf("op.%s.%s[op.oldInt] = op.%s", recv, field, oldF))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s int, elem %s) {\n", r, structName, names.SetAt, idx, elem)
	fmt.Fprintf(b, "\told := %s[%s]\n", acc, idx)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: old, oldInt: %s})\n", kindSet, ub.recvArg(structName, r), oldF, idx)
	fmt.Fprintf(b, "\t%s[%s] = elem\n}\n\n", acc, idx)

	snap := ub.snapField(elem)
	kindRemove := ub.kind(structName, field, "SliceRestore",
		fmt.Sprintf("op.%s.%s = append([]%s(nil), op.%s...)", recv, field, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s int) {\n", r, structName, names.RemoveAt, idx)
	fmt.Fprintf(b, "\toldLen := len(%s)\n", acc)
	// 快照与结果各自独立数组：移位不触碰原底层数组（fork 场景与原根共享），
	// 也不污染存入 undoOp 的快照；make 精确容量避免中途再分配。
	fmt.Fprintf(b, "\tsnap := append([]%s(nil), %s...)\n", elem, acc)
	fmt.Fprintf(b, "\t%s = make([]%s, 0, oldLen-1)\n", acc, elem)
	fmt.Fprintf(b, "\t%s = append(%s, snap[:%s]...)\n", acc, acc, idx)
	fmt.Fprintf(b, "\t%s = append(%s, snap[%s+1:]...)\n", acc, acc, idx)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: snap, oldInt: oldLen})\n", kindRemove, ub.recvArg(structName, r), snap)
	fmt.Fprintf(b, "}\n\n")

	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s int) {\n", r, structName, names.Truncate, truncLen)
	fmt.Fprintf(b, "\tif %s >= len(%s) {\n\t\treturn\n\t}\n", truncLen, acc)
	fmt.Fprintf(b, "\toldLen := len(%s)\n", acc)
	fmt.Fprintf(b, "\t%s = %s[:%s]\n", acc, acc, truncLen)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, oldInt: oldLen})\n", kindTrunc, ub.recvArg(structName, r))
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapSliceOps(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	kp := cowgen.KeyParams(plan.Keys)
	emitStructuredSliceOps(b, ub, structName, r, acc, plan, kp)
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	recv := recvLower(structName)
	keyField := ub.keySlot(plan.Keys[0].KeyType)
	kind := ub.kind(structName, field, "MapSlicePut",
		fmt.Sprintf("if op.had { op.%s.%s[op.%s] = op.%s } else { delete(op.%s.%s, op.%s) }",
			recv, field, keyField, ub.snapField(plan.SliceElem), recv, field, keyField))
	fmt.Fprintf(b, "func (%s *%s) Put%s(ctx *TxContext, %s, val %s) {\n",
		r, structName, plan.FieldName, kp, plan.SliceType)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\told, existed := %s[%s]\n", acc, ka)
	snap := ub.snapField(plan.SliceElem)
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", plan.SliceElem)
	fmt.Fprintf(b, "\t%s[%s] = val\n", acc, ka)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: oldCopy, had: existed})\n",
		kind, ub.recvArg(structName, r), keyField, ka, snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapSliceAppend(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	recv := recvLower(structName)
	keyField := ub.keySlot(plan.Keys[0].KeyType)
	snap := ub.snapField(elem)
	kind := ub.kind(structName, field, "MapSliceAppend",
		fmt.Sprintf("if op.had { op.%s.%s[op.%s] = op.%s[:op.oldInt] } else { delete(op.%s.%s, op.%s) }",
			recv, field, keyField, snap, recv, field, keyField))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, elem %s) {\n", r, structName, names.Append+"At", kp, elem)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\tprev, existed := %s[%s]\n", acc, ka)
	fmt.Fprintf(b, "\toldLen := len(prev)\n")
	fmt.Fprintf(b, "\t%s[%s] = append(prev, elem)\n", acc, ka)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: prev, oldInt: oldLen, had: existed})\n",
		kind, ub.recvArg(structName, r), keyField, ka, snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapSliceSet(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	recv := recvLower(structName)
	idx := indexParamName(r)
	keyField := ub.keySlot(plan.Keys[0].KeyType)
	oldF := ub.leafStoreField(elem)
	kind := ub.kind(structName, field, "MapSliceElemSet",
		fmt.Sprintf("op.%s.%s[op.%s][op.oldInt] = op.%s", recv, field, keyField, oldF))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, %s int, elem %s) {\n", r, structName, names.SetAt, kp, idx, elem)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\toldElem := %s[%s][%s]\n", acc, ka, idx)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: oldElem, oldInt: %s})\n",
		kind, ub.recvArg(structName, r), keyField, ka, oldF, idx)
	fmt.Fprintf(b, "\t%s[%s][%s] = elem\n}\n\n", acc, ka, idx)
}

func emitStructuredMapSliceRemove(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	recv := recvLower(structName)
	idx := indexParamName(r)
	keyField := ub.keySlot(plan.Keys[0].KeyType)
	snap := ub.snapField(elem)
	kind := ub.kind(structName, field, "MapSliceRestore",
		fmt.Sprintf("op.%s.%s[op.%s] = append([]%s(nil), op.%s...)", recv, field, keyField, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, %s int) {\n", r, structName, names.RemoveAt, kp, idx)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\told, existed := %s[%s]\n", acc, ka)
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", elem)
	fmt.Fprintf(b, "\t%s[%s] = append(old[:%s], old[%s+1:]...)\n", acc, ka, idx, idx)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: oldCopy, had: existed})\n",
		kind, ub.recvArg(structName, r), keyField, ka, snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapSliceTruncate(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, kp string) {
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	elem := plan.SliceElem
	recv := recvLower(structName)
	truncLen := truncateLenParamName(r)
	keyField := ub.keySlot(plan.Keys[0].KeyType)
	snap := ub.snapField(elem)
	kind := ub.kind(structName, field, "MapSliceRestore",
		fmt.Sprintf("op.%s.%s[op.%s] = append([]%s(nil), op.%s...)", recv, field, keyField, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, %s int) {\n", r, structName, names.Truncate, kp, truncLen)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\told, existed := %s[%s]\n", acc, ka)
	fmt.Fprintf(b, "\tif %s >= len(old) {\n\t\treturn\n\t}\n", truncLen)
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", elem)
	fmt.Fprintf(b, "\t%s[%s] = old[:%s]\n", acc, ka, truncLen)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: oldCopy, had: existed})\n",
		kind, ub.recvArg(structName, r), keyField, ka, snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapSliceElemGet(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, kp string) {
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	en := plan.ElemName
	if en == "" {
		en = cowgen.Singular(plan.FieldName)
	}
	slot := ptrSlotName(plan.SliceElem)
	recv := recvLower(structName)
	idx := indexParamName(r)
	keyField := ub.keySlot(plan.Keys[0].KeyType)
	kind := ub.kind(structName, field, "MapSlicePtrReplace",
		fmt.Sprintf("op.%s.%s[op.%s][op.oldInt] = op.%s", recv, field, keyField, slot))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, %s int) %s {\n",
		r, structName, cowgen.ElemAtForWriteName(en), kp, idx, plan.SliceElem)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\told := %s[%s][%s]\n", acc, ka, idx)
	fmt.Fprintf(b, "\tif old == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tdirty := old.CloneForWrite()\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: old, oldInt: %s})\n",
		kind, ub.recvArg(structName, r), keyField, ka, slot, idx)
	fmt.Fprintf(b, "\t%s[%s][%s] = dirty\n", acc, ka, idx)
	fmt.Fprintf(b, "\treturn dirty\n}\n\n")
}

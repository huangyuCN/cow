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
	case cowgen.KindMapScalar, cowgen.KindMapStruct:
		emitStructuredMapPut(b, ub, structName, r, acc, plan, false)
	case cowgen.KindMapPtrStruct:
		emitStructuredMapPtrGet(b, ub, structName, r, acc, plan)
		emitStructuredMapPut(b, ub, structName, r, acc, plan, true)
	case cowgen.KindSliceValue, cowgen.KindSlicePtr:
		emitStructuredSliceOps(b, ub, structName, r, acc, plan, "")
	case cowgen.KindMapSliceValue, cowgen.KindMapSlicePtr:
		emitStructuredMapSliceOps(b, ub, structName, r, acc, plan)
	case cowgen.KindMapMapScalar, cowgen.KindMapMapStruct:
		emitStructuredMapMapPut(b, ub, structName, r, acc, plan, false)
		emitStructuredMapMapGetForWrite(b, ub, structName, r, acc, plan)
	case cowgen.KindMapMapPtrStruct:
		emitStructuredMapMapPtrGet(b, ub, structName, r, acc, plan)
		emitStructuredMapMapPut(b, ub, structName, r, acc, plan, true)
		emitStructuredMapMapGetForWrite(b, ub, structName, r, acc, plan)
	case cowgen.KindMapMapSliceValue, cowgen.KindMapMapSlicePtr:
		emitStructuredMapMapSliceOps(b, ub, structName, r, acc, plan)
	}
}

func fieldFromAcc(r, acc string) string {
	return strings.TrimPrefix(acc, r+".")
}

func emitStructuredScalarPut(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := fieldFromAcc(r, acc)
	oldF := oldStoreField(plan.LeafType)
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
	keyField := mapKeyField(plan.Keys[0].KeyType)
	oldF := leafStoreField(plan.LeafType)
	kind := ub.kind(structName, field, "MapKeySet",
		fmt.Sprintf("if op.had { op.%s.%s[op.%s] = op.%s } else { delete(op.%s.%s, op.%s) }",
			recv, field, keyField, oldF, recv, field, keyField))
	fmt.Fprintf(b, "func (%s *%s) Put%s(ctx *TxContext, %s, val %s) {\n", r, structName, plan.FieldName, kp, valType)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+valueTypeForMap(plan))
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
	keyField := mapKeyField(plan.Keys[0].KeyType)
	kind := ub.kind(structName, field, "MapPtrReplace",
		fmt.Sprintf("op.%s.%s[op.%s] = op.%s", recv, field, keyField, slot))
	fmt.Fprintf(b, "func (%s *%s) Get%sForWrite(ctx *TxContext, %s) %s {\n",
		r, structName, plan.ElemName, cowgen.KeyParams(plan.Keys), ret)
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
	kindTrunc := ub.kind(structName, field, "SliceTruncate",
		fmt.Sprintf("op.%s.%s = op.%s.%s[:op.oldInt]", recv, field, recv, field))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, elem %s) {\n", r, structName, names.Append, elem)
	fmt.Fprintf(b, "\toldLen := len(%s)\n", acc)
	fmt.Fprintf(b, "\t%s = append(%s, elem)\n", acc, acc)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, oldInt: oldLen})\n", kindTrunc, ub.recvArg(structName, r))
	fmt.Fprintf(b, "}\n\n")

	oldF := leafStoreField(elem)
	kindSet := ub.kind(structName, field, "SliceSetAt",
		fmt.Sprintf("op.%s.%s[op.oldInt] = op.%s", recv, field, oldF))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, i int, elem %s) {\n", r, structName, names.SetAt, elem)
	fmt.Fprintf(b, "\told := %s[i]\n", acc)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: old, oldInt: i})\n", kindSet, ub.recvArg(structName, r), oldF)
	fmt.Fprintf(b, "\t%s[i] = elem\n}\n\n", acc)

	snap := ub.snapField(elem)
	kindRemove := ub.kind(structName, field, "SliceRestore",
		fmt.Sprintf("op.%s.%s = append([]%s(nil), op.%s...)", recv, field, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, i int) {\n", r, structName, names.RemoveAt)
	fmt.Fprintf(b, "\toldLen := len(%s)\n", acc)
	fmt.Fprintf(b, "\ttail := append([]%s(nil), %s...)\n", elem, acc)
	fmt.Fprintf(b, "\t%s = append(%s[:i], %s[i+1:]...)\n", acc, acc, acc)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: tail, oldInt: oldLen})\n", kindRemove, ub.recvArg(structName, r), snap)
	fmt.Fprintf(b, "}\n\n")

	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, n int) {\n", r, structName, names.Truncate)
	fmt.Fprintf(b, "\tif n >= len(%s) {\n\t\treturn\n\t}\n", acc)
	fmt.Fprintf(b, "\toldLen := len(%s)\n", acc)
	fmt.Fprintf(b, "\t%s = %s[:n]\n", acc, acc)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, oldInt: oldLen})\n", kindTrunc, ub.recvArg(structName, r))
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapSliceOps(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	kp := cowgen.KeyParams(plan.Keys)
	emitStructuredSliceOps(b, ub, structName, r, acc, plan, kp)
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	recv := recvLower(structName)
	keyField := "keyI32"
	if plan.Keys[0].KeyType == "string" {
		keyField = "keyString"
	}
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
	keyField := "keyI32"
	if plan.Keys[0].KeyType == "string" {
		keyField = "keyString"
	}
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
	keyField := "keyI32"
	if plan.Keys[0].KeyType == "string" {
		keyField = "keyString"
	}
	oldF := leafStoreField(elem)
	kind := ub.kind(structName, field, "MapSliceElemSet",
		fmt.Sprintf("op.%s.%s[op.%s][op.oldInt] = op.%s", recv, field, keyField, oldF))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, i int, elem %s) {\n", r, structName, names.SetAt, kp, elem)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\toldElem := %s[%s][i]\n", acc, ka)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: oldElem, oldInt: i})\n",
		kind, ub.recvArg(structName, r), keyField, ka, oldF)
	fmt.Fprintf(b, "\t%s[%s][i] = elem\n}\n\n", acc, ka)
}

func emitStructuredMapSliceRemove(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	recv := recvLower(structName)
	keyField := "keyI32"
	snap := ub.snapField(elem)
	kind := ub.kind(structName, field, "MapSliceRestore",
		fmt.Sprintf("op.%s.%s[op.%s] = append([]%s(nil), op.%s...)", recv, field, keyField, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, i int) {\n", r, structName, names.RemoveAt, kp)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\told, existed := %s[%s]\n", acc, ka)
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", elem)
	fmt.Fprintf(b, "\t%s[%s] = append(old[:i], old[i+1:]...)\n", acc, ka)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: oldCopy, had: existed})\n",
		kind, ub.recvArg(structName, r), keyField, ka, snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapSliceTruncate(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, kp string) {
	field := plan.FieldName
	ka := cowgen.KeyArgs(plan.Keys)
	elem := plan.SliceElem
	recv := recvLower(structName)
	keyField := "keyI32"
	snap := ub.snapField(elem)
	kind := ub.kind(structName, field, "MapSliceRestore",
		fmt.Sprintf("op.%s.%s[op.%s] = append([]%s(nil), op.%s...)", recv, field, keyField, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, n int) {\n", r, structName, names.Truncate, kp)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\told, existed := %s[%s]\n", acc, ka)
	fmt.Fprintf(b, "\tif n >= len(old) {\n\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", elem)
	fmt.Fprintf(b, "\t%s[%s] = old[:n]\n", acc, ka)
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
	keyField := "keyI32"
	kind := ub.kind(structName, field, "MapSlicePtrReplace",
		fmt.Sprintf("op.%s.%s[op.%s][i] = op.%s", recv, field, keyField, slot))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, i int) %s {\n",
		r, structName, cowgen.ElemAtForWriteName(en), kp, plan.SliceElem)
	emitStructuredMapEnsure(b, ub, structName, r, acc, mapTypeString(plan))
	fmt.Fprintf(b, "\told := %s[%s][i]\n", acc, ka)
	fmt.Fprintf(b, "\tif old == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tdirty := old.CloneForWrite()\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: %s, %s: old, oldInt: i})\n",
		kind, ub.recvArg(structName, r), keyField, ka, slot)
	fmt.Fprintf(b, "\t%s[%s][i] = dirty\n", acc, ka)
	fmt.Fprintf(b, "\treturn dirty\n}\n\n")
}

func emitStructuredMapMapPut(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, ptrClone bool) {
	field := plan.FieldName
	kp := cowgen.KeyParams(plan.Keys)
	valType := plan.LeafType
	innerVal := innerValueType(plan)
	recv := recvLower(structName)
	kindOuter := ub.kind(structName, field, "MapMapOuterDelete",
		fmt.Sprintf("delete(op.%s.%s, op.keyI32)", recv, field))
	oldF := leafStoreField(plan.LeafType)
	kindInner := ub.kind(structName, field, "MapMapInnerKeySet",
		fmt.Sprintf(`inner := op.%s.%s[op.keyI32]
if op.had { inner[op.keyString] = op.%s } else { delete(inner, op.keyString) }`, recv, field, oldF))
	fmt.Fprintf(b, "func (%s *%s) Put%s(ctx *TxContext, %s, val %s) {\n", r, structName, plan.FieldName, kp, valType)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner, ok := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif !ok || inner == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOp{kind: %s, %s, had2: true})\n", kindOuter, ub.recvArg(structName, r))
	fmt.Fprintf(b, "\t\tinner = make(map[%s]%s)\n", plan.Keys[1].KeyType, innerVal)
	fmt.Fprintf(b, "\t\t%s[k1] = inner\n", acc)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\told, existed := inner[k2]\n")
	if ptrClone {
		fmt.Fprintf(b, "\tif val != nil {\n\t\tval = val.CloneForWrite()\n\t}\n")
	}
	fmt.Fprintf(b, "\tinner[k2] = val\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, keyI32: k1, keyString: k2, %s: old, had: existed})\n",
		kindInner, ub.recvArg(structName, r), oldF)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapMapGetForWrite(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := plan.FieldName
	kp := cowgen.KeyParams([]cowgen.KeyLayer{plan.Keys[0]})
	recv := recvLower(structName)
	ub.noteInnerMapSnap()
	kindOuter := ub.kind(structName, field, "MapMapOuterRestore",
		fmt.Sprintf("if op.had { op.%s.%s[op.keyI32] = op.innerMapOld } else { delete(op.%s.%s, op.keyI32) }", recv, field, recv, field))
	kindInner := ub.kind(structName, field, "MapMapInnerReplace",
		fmt.Sprintf("op.%s.%s[op.keyI32] = op.innerMapOld", recv, field))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s) map[%s]%s {\n",
		r, structName, cowgen.MapForWriteName(plan.FieldName), kp,
		plan.Keys[1].KeyType, innerValueType(plan))
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\toldInner, existed := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif !existed || oldInner == nil {\n")
	fmt.Fprintf(b, "\t\tnewInner := make(map[%s]%s)\n", plan.Keys[1].KeyType, innerValueType(plan))
	fmt.Fprintf(b, "\t\tctx.push(undoOp{kind: %s, %s, keyI32: k1, had2: !existed})\n", kindOuter, ub.recvArg(structName, r))
	fmt.Fprintf(b, "\t\t%s[k1] = newInner\n", acc)
	fmt.Fprintf(b, "\t\treturn newInner\n\t}\n")
	fmt.Fprintf(b, "\tdirty := clone%sMapShallow(oldInner)\n", plan.FieldName)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, keyI32: k1, innerMapOld: oldInner, had: true})\n", kindInner, ub.recvArg(structName, r))
	fmt.Fprintf(b, "\t%s[k1] = dirty\n", acc)
	fmt.Fprintf(b, "\treturn dirty\n}\n\n")
	emitCloneMapShallow(b, plan)
}

func emitCloneMapShallow(b *bytes.Buffer, plan cowgen.FieldPlan) {
	fmt.Fprintf(b, "func clone%sMapShallow(m map[%s]%s) map[%s]%s {\n",
		plan.FieldName, plan.Keys[1].KeyType, innerValueType(plan),
		plan.Keys[1].KeyType, innerValueType(plan))
	b.WriteString("\tif m == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tc := make(map[%s]%s, len(m))\n", plan.Keys[1].KeyType, innerValueType(plan))
	b.WriteString("\tfor k, v := range m {\n\t\tc[k] = v\n\t}\n")
	b.WriteString("\treturn c\n}\n\n")
}

func emitStructuredMapMapPtrGet(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := plan.FieldName
	ka := cowgen.KeyParams(plan.Keys)
	slot := ptrSlotName(plan.LeafType)
	recv := recvLower(structName)
	kind := ub.kind(structName, field, "MapMapPtrReplace",
		fmt.Sprintf("op.%s.%s[op.keyI32][op.keyString] = op.%s", recv, field, slot))
	fmt.Fprintf(b, "func (%s *%s) Get%sForWrite(ctx *TxContext, %s) %s {\n",
		r, structName, plan.ElemName, ka, plan.LeafType)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif inner == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\told, ok := inner[k2]\n")
	fmt.Fprintf(b, "\tif !ok || old == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tdirty := old.CloneForWrite()\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, keyI32: k1, keyString: k2, %s: old})\n",
		kind, ub.recvArg(structName, r), slot)
	fmt.Fprintf(b, "\tinner[k2] = dirty\n")
	fmt.Fprintf(b, "\treturn dirty\n}\n\n")
}

func emitStructuredMapMapSliceOps(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	kp := cowgen.KeyParams(plan.Keys)
	names := cowgen.SliceMethodNames(plan.FieldName)
	elem := plan.SliceElem
	emitStructuredMapMapSliceAppend(b, ub, structName, r, acc, plan, names, elem, kp)
	emitStructuredMapMapSliceSet(b, ub, structName, r, acc, plan, names, elem, kp)
	emitStructuredMapMapSliceRemove(b, ub, structName, r, acc, plan, names, elem, kp)
	emitStructuredMapMapSliceTruncate(b, ub, structName, r, acc, plan, names, kp)
	field := plan.FieldName
	recv := recvLower(structName)
	snap := ub.snapField(plan.SliceType)
	kindOuter := ub.kind(structName, field, "MapMapSliceOuterDelete",
		fmt.Sprintf("delete(op.%s.%s, op.keyI32)", recv, field))
	kindInner := ub.kind(structName, field, "MapMapSlicePut",
		fmt.Sprintf(`inner := op.%s.%s[op.keyI32]
if op.had { inner[op.keyString] = append([]%s(nil), op.%s...) } else { delete(inner, op.keyString) }`, recv, field, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) Put%s(ctx *TxContext, %s, val %s) {\n",
		r, structName, plan.FieldName, kp, plan.SliceType)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner, ok := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif !ok || inner == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOp{kind: %s, %s, had2: true})\n", kindOuter, ub.recvArg(structName, r))
	fmt.Fprintf(b, "\t\tinner = make(map[%s]%s)\n", plan.Keys[1].KeyType, plan.SliceType)
	fmt.Fprintf(b, "\t\t%s[k1] = inner\n", acc)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\told, existed := inner[k2]\n")
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", elem)
	fmt.Fprintf(b, "\tinner[k2] = val\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, keyI32: k1, keyString: k2, %s: oldCopy, had: existed})\n",
		kindInner, ub.recvArg(structName, r), snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapMapSliceAppend(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	recv := recvLower(structName)
	snap := ub.snapField(elem)
	kindOuter := ub.kind(structName, field, "MapMapSliceOuterDelete",
		fmt.Sprintf("delete(op.%s.%s, op.keyI32)", recv, field))
	kindInner := ub.kind(structName, field, "MapMapSliceAppend",
		fmt.Sprintf(`inner := op.%s.%s[op.keyI32]
if op.had { inner[op.keyString] = op.%s[:op.oldInt] } else { delete(inner, op.keyString) }`, recv, field, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, elem %s) {\n", r, structName, names.Append+"At", kp, elem)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif inner == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOp{kind: %s, %s, had2: true})\n", kindOuter, ub.recvArg(structName, r))
	fmt.Fprintf(b, "\t\tinner = make(map[%s]%s)\n", plan.Keys[1].KeyType, plan.SliceType)
	fmt.Fprintf(b, "\t\t%s[k1] = inner\n", acc)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tprev, existed := inner[k2]\n")
	fmt.Fprintf(b, "\toldLen := len(prev)\n")
	fmt.Fprintf(b, "\tinner[k2] = append(prev, elem)\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, keyI32: k1, keyString: k2, %s: prev, oldInt: oldLen, had: existed})\n",
		kindInner, ub.recvArg(structName, r), snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapMapSliceSet(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	slot := ptrSlotName(elem)
	recv := recvLower(structName)
	kind := ub.kind(structName, field, "MapMapSliceElemSet",
		fmt.Sprintf("op.%s.%s[op.keyI32][op.keyString][i] = op.%s", recv, field, slot))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, i int, elem %s) {\n", r, structName, names.SetAt, kp, elem)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner := %s[k1]\n", acc)
	fmt.Fprintf(b, "\told := inner[k2][i]\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, keyI32: k1, keyString: k2, %s: old, oldInt: i})\n",
		kind, ub.recvArg(structName, r), slot)
	fmt.Fprintf(b, "\tinner[k2][i] = elem\n}\n\n")
}

func emitStructuredMapMapSliceRemove(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	recv := recvLower(structName)
	snap := ub.snapField(elem)
	kind := ub.kind(structName, field, "MapMapSliceRestore",
		fmt.Sprintf("op.%s.%s[op.keyI32][op.keyString] = append([]%s(nil), op.%s...)", recv, field, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, i int) {\n", r, structName, names.RemoveAt, kp)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner := %s[k1]\n", acc)
	fmt.Fprintf(b, "\ts := inner[k2]\n")
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), s...)\n", elem)
	fmt.Fprintf(b, "\tinner[k2] = append(s[:i], s[i+1:]...)\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, keyI32: k1, keyString: k2, %s: oldCopy})\n",
		kind, ub.recvArg(structName, r), snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapMapSliceTruncate(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, kp string) {
	field := plan.FieldName
	elem := plan.SliceElem
	recv := recvLower(structName)
	snap := ub.snapField(elem)
	kind := ub.kind(structName, field, "MapMapSliceRestore",
		fmt.Sprintf("op.%s.%s[op.keyI32][op.keyString] = append([]%s(nil), op.%s...)", recv, field, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, n int) {\n", r, structName, names.Truncate, kp)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner := %s[k1]\n", acc)
	fmt.Fprintf(b, "\ts := inner[k2]\n")
	fmt.Fprintf(b, "\tif n >= len(s) {\n\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), s...)\n", elem)
	fmt.Fprintf(b, "\tinner[k2] = s[:n]\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, keyI32: k1, keyString: k2, %s: oldCopy})\n",
		kind, ub.recvArg(structName, r), snap)
	fmt.Fprintf(b, "}\n\n")
}

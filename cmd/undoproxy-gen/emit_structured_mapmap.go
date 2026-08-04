package main

import (
	"bytes"
	"fmt"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func emitStructuredMapMapPut(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, ptrClone bool) {
	field := plan.FieldName
	kp := cowgen.KeyParams(plan.Keys)
	valType := plan.LeafType
	innerVal := innerValueType(plan)
	recv := recvLower(structName)
	ks := ub.keySlotsFor(plan)
	k1, k2 := ks[0], ks[1]
	// 外层 key 可能「存在但值为 nil map」，回滚须按 had 恢复 nil 槽而非一律删除。
	innerTy := "map[" + plan.Keys[1].KeyType + "]" + innerValueType(plan)
	kindOuter := ub.kind(structName, field, "MapMapOuterRestore",
		fmt.Sprintf("if op.had { op.%s.%s[op.%s] = op.%s } else { delete(op.%s.%s, op.%s) }",
			recv, field, k1, ub.innerMapSlot(innerTy), recv, field, k1))
	oldF := ub.leafStoreField(plan.LeafType)
	kindInner := ub.kind(structName, field, "MapMapInnerKeySet",
		fmt.Sprintf(`inner := op.%s.%s[op.%s]
if op.had { inner[op.%s] = op.%s } else { delete(inner, op.%s) }`, recv, field, k1, k2, oldF, k2))
	fmt.Fprintf(b, "func (%s *%s) Put%s(ctx *TxContext, %s, val %s) {\n", r, structName, plan.FieldName, kp, valType)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner, ok := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif !ok || inner == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOp{kind: %s, %s, %s: k1, had: ok})\n", kindOuter, ub.recvArg(structName, r), k1)
	fmt.Fprintf(b, "\t\tinner = make(map[%s]%s)\n", plan.Keys[1].KeyType, innerVal)
	fmt.Fprintf(b, "\t\t%s[k1] = inner\n", acc)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\told, existed := inner[k2]\n")
	if ptrClone {
		fmt.Fprintf(b, "\tif val != nil {\n\t\tval = val.CloneForWrite()\n\t}\n")
	}
	fmt.Fprintf(b, "\tinner[k2] = val\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: k1, %s: k2, %s: old, had: existed})\n",
		kindInner, ub.recvArg(structName, r), k1, k2, oldF)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapMapGetForWrite(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := plan.FieldName
	kp := cowgen.KeyParams([]cowgen.KeyLayer{plan.Keys[0]})
	recv := recvLower(structName)
	k1 := ub.keySlot(plan.Keys[0].KeyType)
	innerTy := "map[" + plan.Keys[1].KeyType + "]" + innerValueType(plan)
	innerSlot := ub.innerMapSlot(innerTy)
	kindOuter := ub.kind(structName, field, "MapMapOuterRestore",
		fmt.Sprintf("if op.had { op.%s.%s[op.%s] = op.%s } else { delete(op.%s.%s, op.%s) }",
			recv, field, k1, innerSlot, recv, field, k1))
	kindInner := ub.kind(structName, field, "MapMapInnerReplace",
		fmt.Sprintf("op.%s.%s[op.%s] = op.%s", recv, field, k1, innerSlot))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s) map[%s]%s {\n",
		r, structName, cowgen.MapForWriteName(plan.FieldName), kp,
		plan.Keys[1].KeyType, innerValueType(plan))
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\toldInner, existed := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif !existed || oldInner == nil {\n")
	fmt.Fprintf(b, "\t\tnewInner := make(map[%s]%s)\n", plan.Keys[1].KeyType, innerValueType(plan))
	fmt.Fprintf(b, "\t\tctx.push(undoOp{kind: %s, %s, %s: k1, had: existed})\n", kindOuter, ub.recvArg(structName, r), k1)
	fmt.Fprintf(b, "\t\t%s[k1] = newInner\n", acc)
	fmt.Fprintf(b, "\t\treturn newInner\n\t}\n")
	name := cloneMapShallowFuncName(plan.Keys[1].KeyType, innerValueType(plan))
	fmt.Fprintf(b, "\tdirty := %s(oldInner)\n", name)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: k1, %s: oldInner, had: true})\n",
		kindInner, ub.recvArg(structName, r), k1, innerSlot)
	fmt.Fprintf(b, "\t%s[k1] = dirty\n", acc)
	fmt.Fprintf(b, "\treturn dirty\n}\n\n")
	emitCloneMapShallow(b, ub, plan)
}

func emitCloneMapShallow(b *bytes.Buffer, ub *undoBuilder, plan cowgen.FieldPlan) {
	keyType := plan.Keys[1].KeyType
	elemType := innerValueType(plan)
	name := cloneMapShallowFuncName(keyType, elemType)
	if _, ok := ub.cloneHelpers[name]; ok {
		return
	}
	ub.cloneHelpers[name] = struct{}{}
	fmt.Fprintf(b, "func %s(m map[%s]%s) map[%s]%s {\n",
		name, keyType, elemType, keyType, elemType)
	b.WriteString("\tif m == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tc := make(map[%s]%s, len(m))\n", keyType, elemType)
	b.WriteString("\tfor k, v := range m {\n\t\tc[k] = v\n\t}\n")
	b.WriteString("\treturn c\n}\n\n")
}

func emitStructuredMapMapPtrGet(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan) {
	field := plan.FieldName
	ka := cowgen.KeyParams(plan.Keys)
	slot := ptrSlotName(plan.LeafType)
	recv := recvLower(structName)
	ks := ub.keySlotsFor(plan)
	k1, k2 := ks[0], ks[1]
	kind := ub.kind(structName, field, "MapMapPtrReplace",
		fmt.Sprintf("op.%s.%s[op.%s][op.%s] = op.%s", recv, field, k1, k2, slot))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s) %s {\n",
		r, structName, cowgen.MapKeyGetForWriteName(plan.FieldName), ka, plan.LeafType)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif inner == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\told, ok := inner[k2]\n")
	fmt.Fprintf(b, "\tif !ok || old == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tdirty := old.CloneForWrite()\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: k1, %s: k2, %s: old})\n",
		kind, ub.recvArg(structName, r), k1, k2, slot)
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
	ks := ub.keySlotsFor(plan)
	k1, k2 := ks[0], ks[1]
	// 外层 key 可能「存在但值为 nil map」，回滚须按 had 恢复 nil 槽而非一律删除。
	outerTy := "map[" + plan.Keys[1].KeyType + "]" + plan.SliceType
	kindOuter := ub.kind(structName, field, "MapMapSliceOuterRestore",
		fmt.Sprintf("if op.had { op.%s.%s[op.%s] = op.%s } else { delete(op.%s.%s, op.%s) }",
			recv, field, k1, ub.innerMapSlot(outerTy), recv, field, k1))
	kindInner := ub.kind(structName, field, "MapMapSlicePut",
		fmt.Sprintf(`inner := op.%s.%s[op.%s]
if op.had { inner[op.%s] = append([]%s(nil), op.%s...) } else { delete(inner, op.%s) }`,
			recv, field, k1, k2, elem, snap, k2))
	fmt.Fprintf(b, "func (%s *%s) Put%s(ctx *TxContext, %s, val %s) {\n",
		r, structName, plan.FieldName, kp, plan.SliceType)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner, ok := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif !ok || inner == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOp{kind: %s, %s, %s: k1, had: ok})\n", kindOuter, ub.recvArg(structName, r), k1)
	fmt.Fprintf(b, "\t\tinner = make(map[%s]%s)\n", plan.Keys[1].KeyType, plan.SliceType)
	fmt.Fprintf(b, "\t\t%s[k1] = inner\n", acc)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\told, existed := inner[k2]\n")
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", elem)
	fmt.Fprintf(b, "\tinner[k2] = val\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: k1, %s: k2, %s: oldCopy, had: existed})\n",
		kindInner, ub.recvArg(structName, r), k1, k2, snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapMapSliceAppend(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	recv := recvLower(structName)
	snap := ub.snapField(elem)
	ks := ub.keySlotsFor(plan)
	k1, k2 := ks[0], ks[1]
	// 外层 key 可能「存在但值为 nil map」，回滚须按 had 恢复 nil 槽而非一律删除。
	outerTy := "map[" + plan.Keys[1].KeyType + "]" + plan.SliceType
	kindOuter := ub.kind(structName, field, "MapMapSliceOuterRestore",
		fmt.Sprintf("if op.had { op.%s.%s[op.%s] = op.%s } else { delete(op.%s.%s, op.%s) }",
			recv, field, k1, ub.innerMapSlot(outerTy), recv, field, k1))
	kindInner := ub.kind(structName, field, "MapMapSliceAppend",
		fmt.Sprintf(`inner := op.%s.%s[op.%s]
if op.had { inner[op.%s] = op.%s[:op.oldInt] } else { delete(inner, op.%s) }`,
			recv, field, k1, k2, snap, k2))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, elem %s) {\n", r, structName, names.Append+"At", kp, elem)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner, existed := %s[k1]\n", acc)
	fmt.Fprintf(b, "\tif inner == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOp{kind: %s, %s, %s: k1, had: existed})\n", kindOuter, ub.recvArg(structName, r), k1)
	fmt.Fprintf(b, "\t\tinner = make(map[%s]%s)\n", plan.Keys[1].KeyType, plan.SliceType)
	fmt.Fprintf(b, "\t\t%s[k1] = inner\n", acc)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tprev, existed := inner[k2]\n")
	fmt.Fprintf(b, "\toldLen := len(prev)\n")
	fmt.Fprintf(b, "\tinner[k2] = append(prev, elem)\n")
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: k1, %s: k2, %s: prev, oldInt: oldLen, had: existed})\n",
		kindInner, ub.recvArg(structName, r), k1, k2, snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapMapSliceSet(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	slot := ptrSlotName(elem)
	recv := recvLower(structName)
	idx := indexParamName(r)
	ks := ub.keySlotsFor(plan)
	k1, k2 := ks[0], ks[1]
	kind := ub.kind(structName, field, "MapMapSliceElemSet",
		fmt.Sprintf("op.%s.%s[op.%s][op.%s][op.oldInt] = op.%s", recv, field, k1, k2, slot))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, %s int, elem %s) {\n", r, structName, names.SetAt, kp, idx, elem)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner := %s[k1]\n", acc)
	fmt.Fprintf(b, "\told := inner[k2][%s]\n", idx)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: k1, %s: k2, %s: old, oldInt: %s})\n",
		kind, ub.recvArg(structName, r), k1, k2, slot, idx)
	fmt.Fprintf(b, "\tinner[k2][%s] = elem\n}\n\n", idx)
}

func emitStructuredMapMapSliceRemove(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, elem, kp string) {
	field := plan.FieldName
	recv := recvLower(structName)
	idx := indexParamName(r)
	snap := ub.snapField(elem)
	ks := ub.keySlotsFor(plan)
	k1, k2 := ks[0], ks[1]
	kind := ub.kind(structName, field, "MapMapSliceRestore",
		fmt.Sprintf("op.%s.%s[op.%s][op.%s] = append([]%s(nil), op.%s...)", recv, field, k1, k2, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, %s int) {\n", r, structName, names.RemoveAt, kp, idx)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner := %s[k1]\n", acc)
	fmt.Fprintf(b, "\ts := inner[k2]\n")
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), s...)\n", elem)
	fmt.Fprintf(b, "\tinner[k2] = append(s[:%s], s[%s+1:]...)\n", idx, idx)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: k1, %s: k2, %s: oldCopy})\n",
		kind, ub.recvArg(structName, r), k1, k2, snap)
	fmt.Fprintf(b, "}\n\n")
}

func emitStructuredMapMapSliceTruncate(b *bytes.Buffer, ub *undoBuilder, structName, r, acc string, plan cowgen.FieldPlan, names cowgen.SliceMethods, kp string) {
	field := plan.FieldName
	elem := plan.SliceElem
	recv := recvLower(structName)
	truncLen := truncateLenParamName(r)
	snap := ub.snapField(elem)
	ks := ub.keySlotsFor(plan)
	k1, k2 := ks[0], ks[1]
	kind := ub.kind(structName, field, "MapMapSliceRestore",
		fmt.Sprintf("op.%s.%s[op.%s][op.%s] = append([]%s(nil), op.%s...)", recv, field, k1, k2, elem, snap))
	fmt.Fprintf(b, "func (%s *%s) %s(ctx *TxContext, %s, %s int) {\n", r, structName, names.Truncate, kp, truncLen)
	emitStructuredMapEnsure(b, ub, structName, r, acc, "map["+plan.Keys[0].KeyType+"]"+plan.MapValue)
	fmt.Fprintf(b, "\tinner := %s[k1]\n", acc)
	fmt.Fprintf(b, "\ts := inner[k2]\n")
	fmt.Fprintf(b, "\tif %s >= len(s) {\n\t\treturn\n\t}\n", truncLen)
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), s...)\n", elem)
	fmt.Fprintf(b, "\tinner[k2] = s[:%s]\n", truncLen)
	fmt.Fprintf(b, "\tctx.push(undoOp{kind: %s, %s, %s: k1, %s: k2, %s: oldCopy})\n",
		kind, ub.recvArg(structName, r), k1, k2, snap)
	fmt.Fprintf(b, "}\n\n")
}

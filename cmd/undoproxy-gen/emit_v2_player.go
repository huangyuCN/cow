package main

import (
	"bytes"
	"fmt"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func emitPlayerAssetsV2(b *bytes.Buffer, plan cowgen.FieldPlan) {
	fmt.Fprintf(b, "// PutAssetsV2 写 map 字段并记录结构化 Undo。\n")
	fmt.Fprintf(b, "func (p *Player) PutAssetsV2(ctx *TxContextV2, k1 %s, val %s) {\n", plan.Keys[0].KeyType, plan.LeafType)
	fmt.Fprintf(b, "\tif p.Assets == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOpV2{kind: undoKindPlayerAssetsEnsureNilV2, player: p})\n")
	fmt.Fprintf(b, "\t\tp.Assets = make(map[%s]%s)\n", plan.Keys[0].KeyType, plan.LeafType)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\told, existed := p.Assets[k1]\n")
	fmt.Fprintf(b, "\tif existed && old == val {\n\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{\n")
	fmt.Fprintf(b, "\t\tkind:      undoKindPlayerAssetsSetV2,\n")
	fmt.Fprintf(b, "\t\tplayer:    p,\n")
	fmt.Fprintf(b, "\t\tkeyString: k1,\n")
	fmt.Fprintf(b, "\t\toldI64:    old,\n")
	fmt.Fprintf(b, "\t\thad:       existed,\n")
	fmt.Fprintf(b, "\t})\n")
	fmt.Fprintf(b, "\tp.Assets[k1] = val\n")
	fmt.Fprintf(b, "}\n\n")
}

func emitPlayerItemsV2(b *bytes.Buffer, plan cowgen.FieldPlan) {
	elem := plan.SliceElem
	names := cowgen.SliceMethodNames(plan.FieldName)
	fmt.Fprintf(b, "// %sV2 append slice 并记录旧长度。\n", names.Append)
	fmt.Fprintf(b, "func (p *Player) %sV2(ctx *TxContextV2, elem %s) {\n", names.Append, elem)
	fmt.Fprintf(b, "\toldLen := len(p.Items)\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: undoKindPlayerItemsTruncateV2, player: p, oldInt: oldLen})\n")
	fmt.Fprintf(b, "\tp.Items = append(p.Items, elem)\n")
	fmt.Fprintf(b, "}\n\n")

	fmt.Fprintf(b, "// %sV2 删除指定下标元素并记录恢复快照。\n", names.RemoveAt)
	fmt.Fprintf(b, "func (p *Player) %sV2(ctx *TxContextV2, i int) {\n", names.RemoveAt)
	fmt.Fprintf(b, "\toldLen := len(p.Items)\n")
	fmt.Fprintf(b, "\ttail := append([]%s(nil), p.Items...)\n", elem)
	fmt.Fprintf(b, "\tp.Items = append(p.Items[:i], p.Items[i+1:]...)\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: undoKindPlayerItemsSetV2, player: p, oldInt: oldLen, tail: tail})\n")
	fmt.Fprintf(b, "}\n\n")

	fmt.Fprintf(b, "// %sV2 截断切片并记录恢复快照。\n", names.Truncate)
	fmt.Fprintf(b, "func (p *Player) %sV2(ctx *TxContextV2, n int) {\n", names.Truncate)
	fmt.Fprintf(b, "\tif n >= len(p.Items) {\n\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\toldLen := len(p.Items)\n")
	fmt.Fprintf(b, "\ttail := append([]%s(nil), p.Items...)\n", elem)
	fmt.Fprintf(b, "\tp.Items = p.Items[:n]\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: undoKindPlayerItemsSetV2, player: p, oldInt: oldLen, tail: tail})\n")
	fmt.Fprintf(b, "}\n\n")
}

type mapSliceCfgV2 struct {
	mapField     string
	ensureKind   string
	appendKind   string
	setKind      string
	oldFieldName string
	label        string
}

type mapSliceEmitV2 struct {
	plan      cowgen.FieldPlan
	cfg       mapSliceCfgV2
	elem      string
	mapTy     string
	names     cowgen.SliceMethods
	keyParams string
}

func emitPlayerMapSliceV2(b *bytes.Buffer, plan cowgen.FieldPlan, cfg mapSliceCfgV2) {
	emitCtx := mapSliceEmitV2{
		plan:      plan,
		cfg:       cfg,
		elem:      plan.SliceElem,
		mapTy:     mapTypeString(plan),
		names:     cowgen.SliceMethodNames(plan.FieldName),
		keyParams: cowgen.KeyParams(plan.Keys),
	}
	emitPlayerMapSliceAppendV2(b, emitCtx)
	emitPlayerMapSliceSetV2(b, emitCtx)
	emitPlayerMapSliceRemoveV2(b, emitCtx)
	emitPlayerMapSliceTruncateV2(b, emitCtx)
}

func emitEnsureMapBlockV2(b *bytes.Buffer, c mapSliceEmitV2) {
	fmt.Fprintf(b, "\tif %s == nil {\n", c.cfg.mapField)
	fmt.Fprintf(b, "\t\tctx.push(undoOpV2{kind: %s, player: p})\n", c.cfg.ensureKind)
	fmt.Fprintf(b, "\t\t%s = make(%s)\n", c.cfg.mapField, c.mapTy)
	fmt.Fprintf(b, "\t}\n")
}

func emitPlayerMapSliceAppendV2(b *bytes.Buffer, c mapSliceEmitV2) {
	fmt.Fprintf(b, "// %sV2 追加 %s，并记录 k1 对应旧状态。\n", c.names.Append+"At", c.cfg.label)
	fmt.Fprintf(b, "func (p *Player) %sV2(ctx *TxContextV2, %s, elem %s) {\n", c.names.Append+"At", c.keyParams, c.elem)
	emitEnsureMapBlockV2(b, c)
	fmt.Fprintf(b, "\told, existed := %s[k1]\n", c.cfg.mapField)
	fmt.Fprintf(b, "\tnext := append(old, elem)\n")
	if c.cfg.appendKind != "" {
		fmt.Fprintf(b, "\toldLen := len(old)\n")
		fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: %s, player: p, keyI32: k1, %s: old, oldInt: oldLen, had: existed})\n", c.cfg.appendKind, c.cfg.oldFieldName)
	} else {
		fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", c.elem)
		fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: %s, player: p, keyI32: k1, %s: oldCopy, had: existed})\n", c.cfg.setKind, c.cfg.oldFieldName)
	}
	fmt.Fprintf(b, "\t%s[k1] = next\n", c.cfg.mapField)
	fmt.Fprintf(b, "}\n\n")
}

func emitPlayerMapSliceSetV2(b *bytes.Buffer, c mapSliceEmitV2) {
	fmt.Fprintf(b, "// %sV2 设置 %s 指定下标元素，并记录 k1 对应旧状态。\n", c.names.SetAt, c.cfg.label)
	fmt.Fprintf(b, "func (p *Player) %sV2(ctx *TxContextV2, %s, i int, elem %s) {\n", c.names.SetAt, c.keyParams, c.elem)
	emitEnsureMapBlockV2(b, c)
	fmt.Fprintf(b, "\told, existed := %s[k1]\n", c.cfg.mapField)
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", c.elem)
	fmt.Fprintf(b, "\tnext := append([]%s(nil), old...)\n", c.elem)
	fmt.Fprintf(b, "\tnext[i] = elem\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: %s, player: p, keyI32: k1, %s: oldCopy, had: existed})\n", c.cfg.setKind, c.cfg.oldFieldName)
	fmt.Fprintf(b, "\t%s[k1] = next\n", c.cfg.mapField)
	fmt.Fprintf(b, "}\n\n")
}

func emitPlayerMapSliceRemoveV2(b *bytes.Buffer, c mapSliceEmitV2) {
	fmt.Fprintf(b, "// %sV2 删除 %s 指定下标元素，并记录 k1 对应旧状态。\n", c.names.RemoveAt, c.cfg.label)
	fmt.Fprintf(b, "func (p *Player) %sV2(ctx *TxContextV2, %s, i int) {\n", c.names.RemoveAt, c.keyParams)
	emitEnsureMapBlockV2(b, c)
	fmt.Fprintf(b, "\told, existed := %s[k1]\n", c.cfg.mapField)
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", c.elem)
	fmt.Fprintf(b, "\tnext := append([]%s(nil), old...)\n", c.elem)
	fmt.Fprintf(b, "\tnext = append(next[:i], next[i+1:]...)\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: %s, player: p, keyI32: k1, %s: oldCopy, had: existed})\n", c.cfg.setKind, c.cfg.oldFieldName)
	fmt.Fprintf(b, "\t%s[k1] = next\n", c.cfg.mapField)
	fmt.Fprintf(b, "}\n\n")
}

func emitPlayerMapSliceTruncateV2(b *bytes.Buffer, c mapSliceEmitV2) {
	fmt.Fprintf(b, "// %sV2 截断 %s，并记录 k1 对应旧状态。\n", c.names.Truncate, c.cfg.label)
	fmt.Fprintf(b, "func (p *Player) %sV2(ctx *TxContextV2, %s, n int) {\n", c.names.Truncate, c.keyParams)
	emitEnsureMapBlockV2(b, c)
	fmt.Fprintf(b, "\told, existed := %s[k1]\n", c.cfg.mapField)
	fmt.Fprintf(b, "\tif n >= len(old) {\n\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\toldCopy := append([]%s(nil), old...)\n", c.elem)
	fmt.Fprintf(b, "\tnext := append([]%s(nil), old[:n]...)\n", c.elem)
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: %s, player: p, keyI32: k1, %s: oldCopy, had: existed})\n", c.cfg.setKind, c.cfg.oldFieldName)
	fmt.Fprintf(b, "\t%s[k1] = next\n", c.cfg.mapField)
	fmt.Fprintf(b, "}\n\n")
}

func emitPlayerStatsV2(b *bytes.Buffer, plan cowgen.FieldPlan) {
	fmt.Fprintf(b, "// PutStatsV2 写 map[k]map[string]int64，并记录 k1 对应旧状态。\n")
	fmt.Fprintf(b, "func (p *Player) PutStatsV2(ctx *TxContextV2, k1 %s, k2 %s, val %s) {\n", plan.Keys[0].KeyType, plan.Keys[1].KeyType, plan.LeafType)
	fmt.Fprintf(b, "\tif p.Stats == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOpV2{kind: undoKindPlayerStatsEnsureNilV2, player: p})\n")
	fmt.Fprintf(b, "\t\tp.Stats = make(map[%s]%s)\n", plan.Keys[0].KeyType, plan.MapValue)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\tinner, existed := p.Stats[k1]\n")
	fmt.Fprintf(b, "\tcreatedInner := false\n")
	fmt.Fprintf(b, "\tif !existed || inner == nil {\n")
	fmt.Fprintf(b, "\t\tinner = make(map[%s]%s)\n", plan.Keys[1].KeyType, plan.LeafType)
	fmt.Fprintf(b, "\t\tp.Stats[k1] = inner\n")
	fmt.Fprintf(b, "\t\tcreatedInner = true\n")
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\told, hadKey := inner[k2]\n")
	fmt.Fprintf(b, "\tif hadKey && old == val {\n")
	fmt.Fprintf(b, "\t\tif createdInner {\n\t\t\tdelete(p.Stats, k1)\n\t\t}\n")
	fmt.Fprintf(b, "\t\treturn\n\t}\n")
	fmt.Fprintf(b, "\tinner[k2] = val\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: undoKindPlayerStatsPutAtKeyV2, player: p, keyI32: k1, keyString: k2, oldI64: old, had: hadKey, had2: createdInner})\n")
	fmt.Fprintf(b, "}\n\n")

	fmt.Fprintf(b, "// GetStatsMapForWriteV2 返回 map[k]string 的可写副本，并记录 k1 对应旧状态。\n")
	fmt.Fprintf(b, "func (p *Player) GetStatsMapForWriteV2(ctx *TxContextV2, k1 %s) map[%s]%s {\n", plan.Keys[0].KeyType, plan.Keys[1].KeyType, plan.LeafType)
	fmt.Fprintf(b, "\tif p.Stats == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOpV2{kind: undoKindPlayerStatsEnsureNilV2, player: p})\n")
	fmt.Fprintf(b, "\t\tp.Stats = make(map[%s]%s)\n", plan.Keys[0].KeyType, plan.MapValue)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\toldInner, existed := p.Stats[k1]\n")
	fmt.Fprintf(b, "\toldCopy := cloneStatsMapShallowV2(oldInner)\n")
	fmt.Fprintf(b, "\tnext := cloneStatsMapShallowV2(oldInner)\n")
	fmt.Fprintf(b, "\tif next == nil {\n\t\tnext = make(map[%s]%s)\n\t}\n", plan.Keys[1].KeyType, plan.LeafType)
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: undoKindPlayerStatsSetAtKeyV2, player: p, keyI32: k1, statsOld: oldCopy, had: existed && oldInner != nil})\n")
	fmt.Fprintf(b, "\tp.Stats[k1] = next\n")
	fmt.Fprintf(b, "\treturn next\n")
	fmt.Fprintf(b, "}\n\n")
}

func emitPlayerHerosV2(b *bytes.Buffer, plan cowgen.FieldPlan) {
	mapTy := mapTypeFromPlan(plan)
	fmt.Fprintf(b, "// GetHeroForWriteV2 返回 map[k]*Hero 的可写副本，并记录 k1 对应旧状态。\n")
	fmt.Fprintf(b, "func (p *Player) GetHeroForWriteV2(ctx *TxContextV2, %s) %s {\n", cowgen.KeyParams(plan.Keys), plan.LeafType)
	fmt.Fprintf(b, "\tif p.Heros == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOpV2{kind: undoKindPlayerHerosEnsureNilV2, player: p})\n")
	fmt.Fprintf(b, "\t\tp.Heros = make(%s)\n", mapTy)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\told, existed := p.Heros[k1]\n")
	fmt.Fprintf(b, "\tif !existed || old == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tdirty := old.CloneForWriteV2()\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: undoKindPlayerHerosSetAtKeyV2, player: p, keyI32: k1, hero: old, had: existed})\n")
	fmt.Fprintf(b, "\tp.Heros[k1] = dirty\n")
	fmt.Fprintf(b, "\treturn dirty\n")
	fmt.Fprintf(b, "}\n\n")

	fmt.Fprintf(b, "// PutHerosV2 写 map[k]*Hero，并记录 k1 对应旧状态。\n")
	fmt.Fprintf(b, "func (p *Player) PutHerosV2(ctx *TxContextV2, %s, val %s) {\n", cowgen.KeyParams(plan.Keys), plan.LeafType)
	fmt.Fprintf(b, "\tif p.Heros == nil {\n")
	fmt.Fprintf(b, "\t\tctx.push(undoOpV2{kind: undoKindPlayerHerosEnsureNilV2, player: p})\n")
	fmt.Fprintf(b, "\t\tp.Heros = make(%s)\n", mapTy)
	fmt.Fprintf(b, "\t}\n")
	fmt.Fprintf(b, "\told, existed := p.Heros[k1]\n")
	fmt.Fprintf(b, "\tif val != nil {\n\t\tval = val.CloneForWriteV2()\n\t}\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: undoKindPlayerHerosSetAtKeyV2, player: p, keyI32: k1, hero: old, had: existed})\n")
	fmt.Fprintf(b, "\tp.Heros[k1] = val\n")
	fmt.Fprintf(b, "}\n\n")
}

func emitPlayerMainHeroV2(b *bytes.Buffer, plan cowgen.FieldPlan) {
	fmt.Fprintf(b, "// GetMainHeroForWriteV2 返回可写 Hero 副本并记录结构化 Undo。\n")
	fmt.Fprintf(b, "func (p *Player) GetMainHeroForWriteV2(ctx *TxContextV2) %s {\n", plan.LeafType)
	fmt.Fprintf(b, "\told := p.MainHero\n")
	fmt.Fprintf(b, "\tif old == nil {\n\t\treturn nil\n\t}\n")
	fmt.Fprintf(b, "\tdirty := old.CloneForWriteV2()\n")
	fmt.Fprintf(b, "\tctx.push(undoOpV2{kind: undoKindPlayerMainHeroSetV2, player: p, hero: old})\n")
	fmt.Fprintf(b, "\tp.MainHero = dirty\n")
	fmt.Fprintf(b, "\treturn dirty\n")
	fmt.Fprintf(b, "}\n\n")
}

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"

	"github.com/huangyuCN/cow/internal/cowgen"
)

func emitV2Runtime(b *bytes.Buffer) {
	emitV2UndoKindAndOp(b)
	emitV2Context(b)
	emitV2Rollback(b)
	emitV2StatsClone(b)
}

func emitV2UndoKindAndOp(b *bytes.Buffer) {
	b.WriteString("type undoKindV2 uint8\n\n")
	b.WriteString("const (\n")
	b.WriteString("\tundoKindPlayerLevelV2 undoKindV2 = iota + 1\n")
	b.WriteString("\tundoKindPlayerAssetsEnsureNilV2\n")
	b.WriteString("\tundoKindPlayerAssetsSetV2\n")
	b.WriteString("\tundoKindPlayerItemsTruncateV2\n")
	b.WriteString("\tundoKindPlayerItemsSetV2\n")
	b.WriteString("\tundoKindPlayerBagsEnsureNilV2\n")
	b.WriteString("\tundoKindPlayerBagsAppendAtKeyV2\n")
	b.WriteString("\tundoKindPlayerBagsSetAtKeyV2\n")
	b.WriteString("\tundoKindPlayerStatsEnsureNilV2\n")
	b.WriteString("\tundoKindPlayerStatsPutAtKeyV2\n")
	b.WriteString("\tundoKindPlayerStatsSetAtKeyV2\n")
	b.WriteString("\tundoKindPlayerCooldownsEnsureNilV2\n")
	b.WriteString("\tundoKindPlayerCooldownsSetAtKeyV2\n")
	b.WriteString("\tundoKindPlayerHerosEnsureNilV2\n")
	b.WriteString("\tundoKindPlayerHerosSetAtKeyV2\n")
	b.WriteString("\tundoKindPlayerMainHeroSetV2\n")
	b.WriteString("\tundoKindHeroLevelV2\n")
	b.WriteString(")\n\n")

	b.WriteString("type undoOpV2 struct {\n")
	b.WriteString("\tkind undoKindV2\n\n")
	b.WriteString("\tplayer *Player\n")
	b.WriteString("\thero   *Hero\n\n")
	b.WriteString("\tkeyI32    int32\n")
	b.WriteString("\tkeyString string\n\n")
	b.WriteString("\toldI32 int32\n")
	b.WriteString("\toldI64 int64\n")
	b.WriteString("\toldInt int\n\n")
	b.WriteString("\ttail     []*Item\n")
	b.WriteString("\tbagOld   []*Item\n")
	b.WriteString("\tstatsOld map[string]int64\n")
	b.WriteString("\tcdOld    []int32\n\n")
	b.WriteString("\thad  bool\n")
	b.WriteString("\thad2 bool\n")
	b.WriteString("}\n\n")
}

func emitV2Context(b *bytes.Buffer) {
	b.WriteString("// TxContextV2 实验性 Undo 日志（单协程，无锁）。\n")
	b.WriteString("//\n")
	b.WriteString("// 目标：并行验证结构化日志是否降低闭包分配。\n")
	b.WriteString("//\n")
	b.WriteString("// +k8s:deepcopy-gen=false\n")
	b.WriteString("type TxContextV2 struct {\n")
	b.WriteString("\tops []undoOpV2\n")
	b.WriteString("}\n\n")

	b.WriteString("func (ctx *TxContextV2) push(op undoOpV2) {\n")
	b.WriteString("\tctx.ops = append(ctx.ops, op)\n")
	b.WriteString("}\n\n")

	b.WriteString("// Reset 清空日志并复用底层切片。\n")
	b.WriteString("func (ctx *TxContextV2) Reset() {\n")
	b.WriteString("\tctx.ops = ctx.ops[:0]\n")
	b.WriteString("}\n\n")

	b.WriteString("var txPoolV2 = sync.Pool{\n")
	b.WriteString("\tNew: func() any {\n")
	b.WriteString("\t\treturn &TxContextV2{ops: make([]undoOpV2, 0, 16)}\n")
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")
}

func emitV2Rollback(b *bytes.Buffer) {
	b.WriteString("// Rollback 倒序执行所有逆操作。\n")
	b.WriteString("func (ctx *TxContextV2) Rollback() {\n")
	b.WriteString("\tfor i := len(ctx.ops) - 1; i >= 0; i-- {\n")
	b.WriteString("\t\top := ctx.ops[i]\n")
	b.WriteString("\t\tswitch op.kind {\n")
	b.WriteString("\t\tcase undoKindPlayerLevelV2:\n")
	b.WriteString("\t\t\top.player.Level = op.oldI32\n")
	b.WriteString("\t\tcase undoKindPlayerAssetsEnsureNilV2:\n")
	b.WriteString("\t\t\top.player.Assets = nil\n")
	b.WriteString("\t\tcase undoKindPlayerAssetsSetV2:\n")
	b.WriteString("\t\t\tif op.had {\n")
	b.WriteString("\t\t\t\top.player.Assets[op.keyString] = op.oldI64\n")
	b.WriteString("\t\t\t} else {\n")
	b.WriteString("\t\t\t\tdelete(op.player.Assets, op.keyString)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\tcase undoKindPlayerItemsTruncateV2:\n")
	b.WriteString("\t\t\top.player.Items = op.player.Items[:op.oldInt]\n")
	b.WriteString("\t\tcase undoKindPlayerItemsSetV2:\n")
	b.WriteString("\t\t\top.player.Items = append([]*Item(nil), op.tail...)\n")
	b.WriteString("\t\tcase undoKindPlayerBagsEnsureNilV2:\n")
	b.WriteString("\t\t\top.player.Bags = nil\n")
	b.WriteString("\t\tcase undoKindPlayerBagsAppendAtKeyV2:\n")
	b.WriteString("\t\t\tif op.had {\n")
	b.WriteString("\t\t\t\top.player.Bags[op.keyI32] = op.bagOld[:op.oldInt]\n")
	b.WriteString("\t\t\t} else {\n")
	b.WriteString("\t\t\t\tdelete(op.player.Bags, op.keyI32)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\tcase undoKindPlayerBagsSetAtKeyV2:\n")
	b.WriteString("\t\t\tif op.had {\n")
	b.WriteString("\t\t\t\top.player.Bags[op.keyI32] = append([]*Item(nil), op.bagOld...)\n")
	b.WriteString("\t\t\t} else {\n")
	b.WriteString("\t\t\t\tdelete(op.player.Bags, op.keyI32)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\tcase undoKindPlayerStatsEnsureNilV2:\n")
	b.WriteString("\t\t\top.player.Stats = nil\n")
	b.WriteString("\t\tcase undoKindPlayerStatsPutAtKeyV2:\n")
	b.WriteString("\t\t\tinner := op.player.Stats[op.keyI32]\n")
	b.WriteString("\t\t\tif op.had {\n")
	b.WriteString("\t\t\t\tinner[op.keyString] = op.oldI64\n")
	b.WriteString("\t\t\t} else {\n")
	b.WriteString("\t\t\t\tdelete(inner, op.keyString)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\tif op.had2 {\n")
	b.WriteString("\t\t\t\tdelete(op.player.Stats, op.keyI32)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\tcase undoKindPlayerStatsSetAtKeyV2:\n")
	b.WriteString("\t\t\tif op.had {\n")
	b.WriteString("\t\t\t\top.player.Stats[op.keyI32] = cloneStatsMapShallowV2(op.statsOld)\n")
	b.WriteString("\t\t\t} else {\n")
	b.WriteString("\t\t\t\tdelete(op.player.Stats, op.keyI32)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\tcase undoKindPlayerCooldownsEnsureNilV2:\n")
	b.WriteString("\t\t\top.player.Cooldowns = nil\n")
	b.WriteString("\t\tcase undoKindPlayerCooldownsSetAtKeyV2:\n")
	b.WriteString("\t\t\tif op.had {\n")
	b.WriteString("\t\t\t\top.player.Cooldowns[op.keyI32] = append([]int32(nil), op.cdOld...)\n")
	b.WriteString("\t\t\t} else {\n")
	b.WriteString("\t\t\t\tdelete(op.player.Cooldowns, op.keyI32)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\tcase undoKindPlayerHerosEnsureNilV2:\n")
	b.WriteString("\t\t\top.player.Heros = nil\n")
	b.WriteString("\t\tcase undoKindPlayerHerosSetAtKeyV2:\n")
	b.WriteString("\t\t\tif op.had {\n")
	b.WriteString("\t\t\t\top.player.Heros[op.keyI32] = op.hero\n")
	b.WriteString("\t\t\t} else {\n")
	b.WriteString("\t\t\t\tdelete(op.player.Heros, op.keyI32)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\tcase undoKindPlayerMainHeroSetV2:\n")
	b.WriteString("\t\t\top.player.MainHero = op.hero\n")
	b.WriteString("\t\tcase undoKindHeroLevelV2:\n")
	b.WriteString("\t\t\top.hero.Level = op.oldI32\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
}

func emitV2StatsClone(b *bytes.Buffer) {
	b.WriteString("func cloneStatsMapShallowV2(m map[string]int64) map[string]int64 {\n")
	b.WriteString("\tif m == nil {\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("\tout := make(map[string]int64, len(m))\n")
	b.WriteString("\tfor k, v := range m {\n")
	b.WriteString("\t\tout[k] = v\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn out\n")
	b.WriteString("}\n\n")
}

func emitV2FromGraph(output, pkgName string, g *cowgen.Graph) error {
	var b bytes.Buffer
	b.WriteString("// Code generated by undoproxy-gen (mode=v2). DO NOT EDIT.\n\n")
	b.WriteString(fmt.Sprintf("package %s\n\n", pkgName))
	b.WriteString("import \"sync\"\n\n")
	emitV2Runtime(&b)
	for _, sp := range g.Structs {
		switch sp.Name {
		case "Hero":
			emitCloneV2(&b, sp)
			emitHeroMethodsV2(&b, sp)
		case "Player":
			emitPlayerMethodsV2(&b, sp)
		}
	}
	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return fmt.Errorf("format v2 graph: %w", err)
	}
	return os.WriteFile(output, formatted, 0o644)
}

func emitCloneV2(b *bytes.Buffer, sp *cowgen.StructPlan) {
	r := cowgen.RecvIdent(sp.Name)
	fmt.Fprintf(b, "// CloneForWriteV2 返回 %s 的可写浅拷贝。\n", sp.Name)
	fmt.Fprintf(b, "func (%s *%s) CloneForWriteV2() *%s {\n", r, sp.Name, sp.Name)
	fmt.Fprintf(b, "\tif %s == nil {\n\t\treturn nil\n\t}\n", r)
	fmt.Fprintf(b, "\treturn &%s{\n", sp.Name)
	st := sp.Struct
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}
		fmt.Fprintf(b, "\t\t%s: %s.%s,\n", f.Name(), r, f.Name())
	}
	b.WriteString("\t}\n}\n\n")
}

func emitHeroMethodsV2(b *bytes.Buffer, sp *cowgen.StructPlan) {
	r := cowgen.RecvIdent(sp.Name)
	for _, plan := range sp.Plans {
		if plan.Kind != cowgen.KindScalar || plan.FieldName != "Level" {
			continue
		}
		fmt.Fprintf(b, "// PutLevelV2 写 Hero.Level 并记录结构化 Undo。\n")
		fmt.Fprintf(b, "func (%s *Hero) PutLevelV2(ctx *TxContextV2, val %s) {\n", r, plan.LeafType)
		fmt.Fprintf(b, "\told := %s.%s\n", r, plan.FieldName)
		fmt.Fprintf(b, "\tif old == val {\n\t\treturn\n\t}\n")
		fmt.Fprintf(b, "\tctx.push(undoOpV2{\n")
		fmt.Fprintf(b, "\t\tkind:   undoKindHeroLevelV2,\n")
		fmt.Fprintf(b, "\t\thero:   %s,\n", r)
		fmt.Fprintf(b, "\t\toldI32: old,\n")
		fmt.Fprintf(b, "\t})\n")
		fmt.Fprintf(b, "\t%s.%s = val\n", r, plan.FieldName)
		fmt.Fprintf(b, "}\n\n")
	}
}

func emitPlayerMethodsV2(b *bytes.Buffer, sp *cowgen.StructPlan) {
	for _, plan := range sp.Plans {
		switch plan.FieldName {
		case "Assets":
			emitPlayerAssetsV2(b, plan)
		case "Items":
			emitPlayerItemsV2(b, plan)
		case "Bags":
			emitPlayerMapSliceV2(b, plan, mapSliceCfgV2{
				mapField:     "p.Bags",
				ensureKind:   "undoKindPlayerBagsEnsureNilV2",
				appendKind:   "undoKindPlayerBagsAppendAtKeyV2",
				setKind:      "undoKindPlayerBagsSetAtKeyV2",
				oldFieldName: "bagOld",
				label:        "map[k][]*Item",
			})
		case "Stats":
			emitPlayerStatsV2(b, plan)
		case "Cooldowns":
			emitPlayerMapSliceV2(b, plan, mapSliceCfgV2{
				mapField:     "p.Cooldowns",
				ensureKind:   "undoKindPlayerCooldownsEnsureNilV2",
				appendKind:   "",
				setKind:      "undoKindPlayerCooldownsSetAtKeyV2",
				oldFieldName: "cdOld",
				label:        "map[k][]int32",
			})
		case "Heros":
			emitPlayerHerosV2(b, plan)
		case "MainHero":
			emitPlayerMainHeroV2(b, plan)
		}
	}
}

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

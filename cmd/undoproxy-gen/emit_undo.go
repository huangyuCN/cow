package main

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"github.com/huangyuCN/cow/internal/cowgen"
)

type undoEntry struct {
	name string
	body string
}

// undoBuilder 收集结构化 Undo 的 kind 与 Rollback 分支，并驱动运行时代码生成。
type undoBuilder struct {
	structs      []string
	entries      []undoEntry
	kindIndex    map[string]string
	sliceSnaps   map[string]string // slice 元素类型 -> undoOp 字段名
	innerMapSnap bool              // 是否需要 statsOld map[string]int64
}

func newUndoBuilder(g *cowgen.Graph) *undoBuilder {
	seen := make(map[string]struct{})
	var names []string
	for _, sp := range g.Structs {
		n := sp.Name
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		names = append(names, n)
	}
	return &undoBuilder{
		structs:    names,
		kindIndex:  make(map[string]string),
		sliceSnaps: make(map[string]string),
	}
}

func kindName(structName, field, op string) string {
	return "undoKind" + structName + field + op
}

func (ub *undoBuilder) kind(structName, field, op, rollbackBody string) string {
	key := structName + "." + field + "." + op
	if n, ok := ub.kindIndex[key]; ok {
		return n
	}
	n := kindName(structName, field, op)
	ub.kindIndex[key] = n
	ub.entries = append(ub.entries, undoEntry{name: n, body: rollbackBody})
	return n
}

func (ub *undoBuilder) recvArg(structName, recv string) string {
	return fmt.Sprintf("%s: %s", recvLower(structName), recv)
}

// snapField 为 slice 快照注册 undoOp 字段（按元素类型去重）。
func (ub *undoBuilder) snapField(elemType string) string {
	if name, ok := ub.sliceSnaps[elemType]; ok {
		return name
	}
	base := snapFieldBase(elemType)
	name := base
	for i := 2; ub.sliceFieldUsed(name); i++ {
		name = fmt.Sprintf("%s%d", base, i)
	}
	ub.sliceSnaps[elemType] = name
	return name
}

func snapFieldBase(elemType string) string {
	var b strings.Builder
	b.WriteString("snap")
	for _, r := range elemType {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (ub *undoBuilder) sliceFieldUsed(name string) bool {
	for _, v := range ub.sliceSnaps {
		if v == name {
			return true
		}
	}
	return false
}

func (ub *undoBuilder) noteInnerMapSnap() {
	ub.innerMapSnap = true
}

func (ub *undoBuilder) writeRuntime(b *bytes.Buffer) {
	b.WriteString("type undoKind uint8\n\nconst (\n")
	for i, e := range ub.entries {
		if i == 0 {
			fmt.Fprintf(b, "\t%s undoKind = iota + 1\n", e.name)
		} else {
			fmt.Fprintf(b, "\t%s\n", e.name)
		}
	}
	b.WriteString(")\n\n")

	b.WriteString("type undoOp struct {\n\tkind undoKind\n")
	for _, sn := range ub.structs {
		fmt.Fprintf(b, "\t%s *%s\n", recvLower(sn), sn)
	}
	b.WriteString("\tkeyI32    int32\n")
	b.WriteString("\tkeyI64    int64\n")
	b.WriteString("\tkeyU64    uint64\n")
	b.WriteString("\tkeyString string\n\n")
	b.WriteString("\toldI32    int32\n")
	b.WriteString("\toldI64    int64\n")
	b.WriteString("\toldU64    uint64\n")
	b.WriteString("\toldInt    int\n")
	b.WriteString("\toldString string\n\n")
	for _, elemType := range ub.sortedSliceTypes() {
		fmt.Fprintf(b, "\t%s []%s\n", ub.sliceSnaps[elemType], elemType)
	}
	if ub.innerMapSnap {
		b.WriteString("\tinnerMapOld map[string]int64\n\n")
	}
	b.WriteString("\thad  bool\n")
	b.WriteString("\thad2 bool\n")
	b.WriteString("}\n\n")

	b.WriteString("// TxContext 单次请求作用域的 Undo 日志（单协程，无锁）。\n")
	b.WriteString("//\n")
	b.WriteString("// +k8s:deepcopy-gen=false\n")
	b.WriteString("type TxContext struct {\n\tops []undoOp\n}\n\n")
	b.WriteString("func (ctx *TxContext) push(op undoOp) {\n")
	b.WriteString("\tctx.ops = append(ctx.ops, op)\n")
	b.WriteString("}\n\n")
	b.WriteString("// Reset 清空日志并复用底层切片。\n")
	b.WriteString("func (ctx *TxContext) Reset() {\n")
	b.WriteString("\tfor i := range ctx.ops {\n")
	b.WriteString("\t\tctx.ops[i] = undoOp{}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tctx.ops = ctx.ops[:0]\n")
	b.WriteString("}\n\n")
	b.WriteString("var txPool = sync.Pool{\n")
	b.WriteString("\tNew: func() any {\n")
	b.WriteString("\t\treturn &TxContext{ops: make([]undoOp, 0, 16)}\n")
	b.WriteString("\t},\n")
	b.WriteString("}\n\n")
	b.WriteString("// Rollback 倒序执行所有逆操作。\n")
	b.WriteString("func (ctx *TxContext) Rollback() {\n")
	b.WriteString("\tfor i := len(ctx.ops) - 1; i >= 0; i-- {\n")
	b.WriteString("\t\top := ctx.ops[i]\n")
	b.WriteString("\t\tswitch op.kind {\n")
	for _, e := range ub.entries {
		fmt.Fprintf(b, "\t\tcase %s:\n", e.name)
		for _, line := range strings.Split(strings.TrimSpace(e.body), "\n") {
			fmt.Fprintf(b, "\t\t\t%s\n", line)
		}
	}
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
}

func (ub *undoBuilder) sortedSliceTypes() []string {
	types := make([]string, 0, len(ub.sliceSnaps))
	for t := range ub.sliceSnaps {
		types = append(types, t)
	}
	// 简单字典序，保证生成稳定
	for i := 0; i < len(types); i++ {
		for j := i + 1; j < len(types); j++ {
			if types[j] < types[i] {
				types[i], types[j] = types[j], types[i]
			}
		}
	}
	return types
}

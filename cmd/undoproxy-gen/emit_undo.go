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
	scalarOlds   map[string]string // 标量类型字符串 -> undoOp 旧值字段名
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
	ub := &undoBuilder{
		structs:    names,
		kindIndex:  make(map[string]string),
		sliceSnaps: make(map[string]string),
		scalarOlds: make(map[string]string),
	}
	// slice 下标/长度与标量 int 共用（生成代码中大量 oldInt: i / oldInt: oldLen）
	ub.scalarOlds["int"] = "oldInt"
	return ub
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

// scalarOldField 为标量旧值注册 undoOp 字段（按 Go 类型字符串去重）。
func (ub *undoBuilder) scalarOldField(goType string) string {
	if goType == "" {
		return "oldI64"
	}
	if name, ok := ub.scalarOlds[goType]; ok {
		return name
	}
	name := canonicalScalarOldName(goType)
	for i := 2; ub.scalarOldNameUsed(name); i++ {
		name = fmt.Sprintf("%s%d", canonicalScalarOldName(goType), i)
	}
	ub.scalarOlds[goType] = name
	return name
}

func (ub *undoBuilder) scalarOldNameUsed(name string) bool {
	for _, v := range ub.scalarOlds {
		if v == name {
			return true
		}
	}
	return false
}

// leafStoreField 返回 undoOp 中存放叶子旧值的字段名（标量或 *Struct 指针槽）。
func (ub *undoBuilder) leafStoreField(goType string) string {
	if strings.HasPrefix(goType, "*") {
		return ptrSlotName(goType)
	}
	return ub.scalarOldField(goType)
}

func canonicalScalarOldName(goType string) string {
	switch goType {
	case "int32":
		return "oldI32"
	case "int64":
		return "oldI64"
	case "uint64":
		return "oldU64"
	case "uint32":
		return "oldU32"
	case "uint16":
		return "oldU16"
	case "uint8":
		return "oldU8"
	case "int", "uint", "uintptr":
		return "oldInt"
	case "string":
		return "oldString"
	case "float32":
		return "oldF32"
	case "float64":
		return "oldF64"
	case "bool":
		return "oldBool"
	default:
		return scalarOldNameFromType(goType)
	}
}

func scalarOldNameFromType(goType string) string {
	var b strings.Builder
	b.WriteString("old")
	for _, r := range goType {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if r >= 'a' && r <= 'z' && b.Len() == 3 {
				b.WriteRune(r - ('a' - 'A'))
				continue
			}
			b.WriteRune(r)
		}
	}
	if b.Len() <= 3 {
		return "oldMisc"
	}
	return b.String()
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
	b.WriteString("\tkeyU32    uint32\n")
	b.WriteString("\tkeyU64    uint64\n")
	b.WriteString("\tkeyString string\n\n")
	for _, goType := range ub.sortedScalarOldTypes() {
		fmt.Fprintf(b, "\t%s %s\n", ub.scalarOlds[goType], goType)
	}
	if len(ub.scalarOlds) > 0 {
		b.WriteString("\n")
	}
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

func (ub *undoBuilder) sortedScalarOldTypes() []string {
	return sortedMapKeys(ub.scalarOlds)
}

func sortedMapKeys(m map[string]string) []string {
	types := make([]string, 0, len(m))
	for t := range m {
		types = append(types, t)
	}
	for i := 0; i < len(types); i++ {
		for j := i + 1; j < len(types); j++ {
			if types[j] < types[i] {
				types[i], types[j] = types[j], types[i]
			}
		}
	}
	return types
}

func (ub *undoBuilder) sortedSliceTypes() []string {
	return sortedMapKeys(ub.sliceSnaps)
}

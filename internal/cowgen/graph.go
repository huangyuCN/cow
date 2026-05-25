package cowgen

import (
	"fmt"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/huangyuCN/cow/internal/cowmon"
)

// BuildGraph 从包信息构建 undoproxy 类型图。
func BuildGraph(pkg *cowmon.PackageInfo) (*Graph, error) {
	reachable, err := cowmon.CollectReachable(pkg)
	if err != nil {
		return nil, err
	}
	g := &Graph{}
	for _, named := range reachable {
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		sp := &StructPlan{Name: named.Obj().Name(), Clone: true, Struct: st}
		for i := 0; i < st.NumFields(); i++ {
			f := st.Field(i)
			if !f.Exported() {
				continue
			}
			plan, err := classifyField(f.Type(), pkg.Pkg)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", sp.Name, f.Name(), err)
			}
			if plan != nil {
				plan.FieldName = f.Name()
				sp.Plans = append(sp.Plans, *plan)
			}
		}
		g.Structs = append(g.Structs, sp)
	}
	return g, nil
}

// TypeStr 打印类型字符串（同包省略包名）。
func TypeStr(pkg *types.Package, t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string {
		if p == nil || p == pkg {
			return ""
		}
		return p.Name()
	})
}

// RecvIdent 生成方法 receiver 变量名。
func RecvIdent(structName string) string {
	if structName == "" {
		return "x"
	}
	r, _ := utf8.DecodeRuneInString(structName)
	return string(unicode.ToLower(r))
}

// KeyParams 生成 map key 形参列表。
func KeyParams(keys []KeyLayer) string {
	var ps []string
	for i, k := range keys {
		ps = append(ps, fmt.Sprintf("k%d %s", i+1, k.KeyType))
	}
	return strings.Join(ps, ", ")
}

// KeyArgs 生成 map key 实参列表。
func KeyArgs(keys []KeyLayer) string {
	var as []string
	for i := range keys {
		as = append(as, fmt.Sprintf("k%d", i+1))
	}
	return strings.Join(as, ", ")
}

# undorewrite

**集成用法**：[docs/guide/migration-undorewrite.md](../../docs/guide/migration-undorewrite.md)

## 职责

扫描指定路径下 Go 源码，将监控类型上的**裸写**改为 `undoproxy-gen` 生成的代理调用。默认 **dry-run**；`-w` 写回。使用 `go/types` + `internal/cowproxy` 目录，与生成器字段分类一致。

## 能力边界

- 不改 `zz_generated*`、`*_fixture.go` 等（与 `undocheck` 白名单一致）。
- 不自动改函数签名（除非 `-inject-ctx`）。
- 不替代 `undoproxy-gen` 或 `undocheck`。

## 安装与用法

```bash
go install ./cmd/undorewrite

undorewrite [flags] ./patterns...

# flags（见 main.go）
#   -cow string      cow 模块 import path（默认 github.com/huangyuCN/cow）
#   -w               写回源文件
#   -ctx string      TxContext 变量名（默认 ctx）
#   -inject-ctx      new | pool | param:NAME
#   -pool-var string pool 变量名（默认 txPool）
```

退出码：用法错误 `2`，运行错误 `1`，有跳过/错误汇总时 `1`。

## 典型流程

```bash
undorewrite ./yourpkg/...           # 审查 diff
undorewrite -w ./yourpkg/...
go vet -vettool=$(go env GOPATH)/bin/undocheck ./yourpkg/...
```

## 源码地图

| 文件 | 职责 |
|------|------|
| `main.go` | CLI、打印 summary |
| `config.go` | `Config` 结构 |
| `load.go` | `packages.Load` |
| `rewrite.go` | AST 重写核心 |
| `ctx.go` | ctx 解析与注入 |
| `path.go` | LHS 路径分解 |
| `diff.go` | dry-run diff 输出 |

## 相关链接

- 设计：[docs/superpowers/specs/2026-05-25-undorewrite-codemod-design.md](../../docs/superpowers/specs/2026-05-25-undorewrite-codemod-design.md)
- 守门：[cmd/undocheck/README.md](../undocheck/README.md)

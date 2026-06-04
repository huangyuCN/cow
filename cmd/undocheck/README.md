# undocheck

**集成用法**：[docs/guide/bare-write-guard.md](../../docs/guide/bare-write-guard.md)

## 职责

`go/analysis` 分析器 **`cowbarewrite`**：禁止对 undoproxy 监控类型的裸写，提示改用生成代理方法。

## 检查范围

- **本包**含 `// +cow:undoproxy-gen=true` 根类型时，检查本包非白名单源码。
- **直接 import** 的其它包若含上述根类型，将其类型图并入监控集，检查**当前包**内对这些类型的裸写（无需 `import github.com/huangyuCN/cow` 根模块）。
- **不**沿传递依赖检查（仅 `import` 了中间包、未直接 import 定义包时不查）。

详见 [import-scope 设计](../../docs/superpowers/specs/2026-06-02-undocheck-import-scope-design.md)。

## 能力边界

- 仅静态分析；不改源码。
- 不拦截反射 / `Unmarshal` 写入。
- 字段保持导出；不靠改为非导出字段防裸写。

## 安装与用法

```bash
go install ./cmd/undocheck

go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
# 或
go vet -cowbarewrite ./...
```

也可作为独立 checker 二进制（`main.go` 调用 `singlechecker.Main`）。

## 典型 CI

```bash
go install ./cmd/undocheck
go vet -vettool=$(which undocheck) ./...
```

消费方 import 本模块类型时，须在其仓库重复上述步骤。

## 源码地图

| 文件 | 职责 |
|------|------|
| `analyzer.go` | 注册 `cowbarewrite`、`monitoredForPass`（本包 + direct import 合并类型图） |
| `srcdir.go` | analysistest `testdata/src` 回退加载 |
| `inspect.go` | AST 裸写检测与报告 |
| `whitelist.go` | 委托 `internal/cowfile` 跳过规则 |
| `suggest.go` | 修复建议文案 |
| `testdata/` | 好/坏例 |

监控集合：`internal/cowmon`（与 `undoproxy-gen` 类型图对齐）。

## 相关链接

- 设计：[docs/superpowers/specs/2026-05-25-bare-write-guard-design.md](../../docs/superpowers/specs/2026-05-25-bare-write-guard-design.md)
- 迁移：[docs/guide/migration-undorewrite.md](../../docs/guide/migration-undorewrite.md)

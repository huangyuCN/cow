# examples/gamestore 独立接入示例设计说明

| 项 | 值 |
|---|---|
| 状态 | 已实现（2026-05-26） |
| 路径 | `examples/gamestore/` |
| 场景 | 集成方从零接入：自有聚合根、`go generate`、请求作用域 Undo、静态守门 |
| 前置 | [2026-05-26-undoproxy-gen-structured-generic-design.md](2026-05-26-undoproxy-gen-structured-generic-design.md)、[docs/guide/integration-checklist.md](../../guide/integration-checklist.md) |

## 1. 问题

根包 `cow` 以库自身 `Player` 夹具演示 Undo 与 benchmark，集成方难以区分「库内部测试」与「业务模块应如何搭」。`doc_examples_test.go` 仅覆盖 lite 片段，且位于 `package cow`，无法展示：

- 独立 `go.mod` + `replace` 引用本仓库；
- 业务方自定义类型图与 `//go:generate undoproxy-gen`；
- 双根 `+cow:undoproxy-gen=true`；
- `undocheck` 在消费者包上的用法。

## 2. 目标

1. 在仓库根目录新增 **`examples/gamestore/`**，作为**独立 Go module**（不 import `cow.Player`）。
2. 类型图覆盖 `undoproxy-gen` 主要 `FieldKind`（标量、map、slice、指针、map 指针、map slice、map map），并含 **两个** tag 根类型。
3. 可运行：`go run .` 演示 **Rollback**（失败恢复）与 **Commit**（成功 `Reset`）；`go test ./...` 断言正确性。
4. README 说明 generate、run、vet+`undocheck`；根 `README.md` 链到该示例。
5. CI 对示例子模块执行 `go test` 与 `go build`（不并入根 `go test ./...`）。

## 3. 非目标

- 不复制 mega ~1MB 夹具；示例保持中等规模、可读优先。
- 不在示例内执行 `undorewrite`（仅在 README 链到 [migration-undorewrite.md](../../guide/migration-undorewrite.md)）。
- 不引入 `deepcopy-gen`（运行路径不依赖；benchmark 对照见根包文档）。
- 不新增第二个 example 子模块（YAGNI；后续若有需求再加 `examples/minimal`）。

## 4. 方案选择

| 方案 | 结论 |
|---|---|
| A. 独立消费者模块 `examples/gamestore/` | **采用** |
| B. 薄封装，直接 import `cow.Player` | 不采用 |
| C. minimal + full 双子模块 | 不采用 |

## 5. 目录与模块

```text
examples/gamestore/
  go.mod
  doc.go
  types.go
  generate.go
  zz_generated.undo_proxy.go
  service.go
  handler.go
  main.go
  handler_test.go
  README.md
```

### 5.1 `go.mod`

```go
module github.com/huangyuCN/cow/examples/gamestore

go 1.25

require github.com/huangyuCN/cow v0.0.0

replace github.com/huangyuCN/cow => ../..
```

- 示例包**仅**通过 `replace` 使用本仓库的 **`cmd/undoproxy-gen`**（`go run` / 文档中的 `go install`）；**不** `import` 根包 `github.com/huangyuCN/cow` 的业务类型。
- 生成后的 `TxContext`、`Rollback`、`Put*` 等均在 **`package gamestore`** 的 `zz_generated.undo_proxy.go` 内。

### 5.2 代码生成

`generate.go`：

```go
//go:generate go run github.com/huangyuCN/cow/cmd/undoproxy-gen --output-file zz_generated.undo_proxy.go github.com/huangyuCN/cow/examples/gamestore
```

- `doc.go` 含 `// +cow:undoproxy-gen=package`（与根包约定一致）。
- 生成物 **提交 Git**；README 要求改 `types.go` 后执行 `go generate ./...`。

## 6. 类型图

同包 struct；**双根**均打 `// +cow:undoproxy-gen=true`。

### 6.1 `Player`（主聚合根）

| 字段 | 类型 | 覆盖 Kind |
|------|------|-----------|
| `Gold` | `int64` | 标量 |
| `Wallet` | `map[string]int64` | map 标量 |
| `Items` | `[]*Item` | slice 指针元素 |
| `MainHero` | `*Hero` | 指针 struct |
| `Heros` | `map[int32]*Hero` | map 指针 |
| `Bags` | `map[int32][]*Item` | map + slice |
| `Stats` | `map[int32]map[string]int64` | map map |

### 6.2 `Guild`（第二根）

| 字段 | 类型 | 覆盖 Kind |
|------|------|-----------|
| `Members` | `map[int32]*Member` | map 指针（第二根 BFS 可达） |

### 6.3 嵌套 struct（无 tag，由可达性纳入）

- `Item`：`Id int64`
- `Hero`：`Level int32`，`Skills map[int32]*Skill`
- `Member`：`Name string`，`Rank int32`
- `Skill`：`Level int32`

夹具构造函数 `newDemoPlayer()` / `newDemoGuild()` 填充中等规模 map/slice（如数十项），保证 `Get*ForWrite` 路径非 nil。

## 7. 运行时与业务演示

### 7.1 `service.go`

提供与根包测试同语义的作用域辅助（示例内自包含）：

- `runScopedCommit(fn func(ctx *TxContext) error) error` — 成功时 `Reset()`，失败时 `Rollback()`。
- `runScopedWithRollback(fn func(ctx *TxContext) error) error` — defer 总是 `Rollback()`（用于可恢复性演示/测试）。

均从 `txPool` 取还 `*TxContext`。

### 7.2 `handler.go`

单次请求内 **约 10～15 处**生成代理写，覆盖：

- `PutGold`、`PutWallet`（标量 / map 标量）
- `AppendItems`、`SetItemsAt`（slice）
- `GetMainHeroForWrite` + `PutLevel`（指针 COW）
- `GetHeroForWrite` + 子 struct `Put`
- `PutStats` 或 `GetStatsMapForWrite` 内层写（map map）
- `AppendBagsAt`（map slice）
- `Guild`：`PutMembers` 或 `GetMemberForWrite`（第二根）

提供两个入口：

- `HandlePurchaseSuccess(p *Player, g *Guild, ctx *TxContext) error` — 正常提交路径；
- `HandlePurchaseFail(p *Player, ctx *TxContext) error` — 中途返回 `error`，供 Rollback 演示。

### 7.3 `main.go`

```text
seedP, seedG := newDemoPlayer(), newDemoGuild()
demoRollback(copy)  // 打印/说明回滚后 Gold 等恢复
demoCommit(copy)    // 打印/说明提交后变更保留
```

使用 `DeepCopy` 仅用于 **演示前复制种子**（可选：手写快照函数，避免示例依赖 `deepcopy-gen`；若不用 k8s deepcopy，则用浅拷贝 + 固定字段比较或 `go-cmp` 仅测关键字段）。

**约定**：示例内聚合根状态比较以 **关键字段** + `handler_test` 为准；`main` 以 `fmt.Println` 说明结果，不强制引入 deepcopy-gen。

### 7.4 与当前生成器约定对齐

- 使用 `ctx.push(undoOp{...})` 生成路径；示例代码**不得**调用 `AddUndo`（已移除）。
- `handler_test` 可选断言生成文件不含 `AddUndo` 字符串（防回归）。

## 8. 工具链在示例中的体现

| 能力 | 体现方式 |
|------|----------|
| `undoproxy-gen` | `generate.go` + 已提交 `zz_generated.undo_proxy.go` |
| `TxContext` / Pool / Reset / Rollback | `service.go`、`handler.go` |
| 多根类型图 | `Player` + `Guild` |
| `undocheck` | README：`go install github.com/huangyuCN/cow/cmd/undocheck` + `go vet -vettool=... ./...` |
| `undorewrite` | README 链接迁移文档，不执行 |
| 代理 API 全集（主要 Kind） | `handler.go` 注释表对照 [proxy-api.md](../../guide/proxy-api.md) |

## 9. 测试

| 测试 | 断言 |
|------|------|
| `TestHandlePurchaseFail_Rollback` | Rollback 后 `Player` 关键字段与初始快照一致 |
| `TestHandlePurchaseSuccess_Commit` | Commit 后 `Gold`/`Wallet`/Items 等按业务预期变化 |
| `TestGuildMember_Write`（可选） | 第二根 `Guild` 写入可回滚 |
| `TestGenerated_contract`（可选） | 生成物含 `type TxContext`、`func (ctx *TxContext) Rollback`，且无 `AddUndo` |

测试不依赖根包 `cow` 的 `runScoped*` 或 `newBenchPlayer`。

## 10. CI

在 `.github/workflows/ci.yml` 增加 job（或与 `test` 并行）：

```yaml
example-gamestore:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: "1.25.x"
        cache: true
    - run: cd examples/gamestore && go test ./... && go build -o /dev/null .
```

根 job `go test ./...` **不**包含 `examples/`（独立 module）。

可选：对 `examples/gamestore` 跑 `undocheck` vet（与根包 vet job 分开或合并）。

## 11. 文档链接

| 文件 | 变更 |
|------|------|
| `examples/gamestore/README.md` | 新建：快速开始、generate、run、vet |
| `README.md` | 增加「完整示例」链到 `examples/gamestore` |
| `docs/README.md` 或 `docs/guide/README.md` | 集成方索引增加 example 链接（一行） |

## 12. 验收标准

1. `cd examples/gamestore && go test ./... && go run .` 成功。
2. `go generate ./...` 在示例子模块内可再生成且 diff 为空（或仅注释差异）。
3. 示例源码无聚合根裸写；`go vet -vettool=undocheck ./...` 在示例子模块无 `cowbarewrite` 诊断。
4. CI `example-gamestore` job 绿。
5. 类型图含双根；`handler` 覆盖 §6 所列 Kind 的代表性操作。

## 13. 相关链接

- [integration-checklist.md](../../guide/integration-checklist.md)
- [codegen-undoproxy.md](../../guide/codegen-undoproxy.md)
- [tx-context.md](../../guide/tx-context.md)
- [2026-05-25-undoproxy-codegen-design.md](2026-05-25-undoproxy-codegen-design.md)

# examples/gamestore-migrate 裸写迁移示例设计说明

| 项 | 值 |
|---|---|
| 状态 | 已批准（brainstorming 2026-05-27） |
| 路径 | `examples/gamestore-migrate/` |
| 场景 | 独立 module：从存量裸写对照学习，经 tag → generate → undorewrite → undocheck → Tx 运行时，过渡到 cow 写模式 |
| 前置 | [2026-05-26-examples-gamestore-design.md](2026-05-26-examples-gamestore-design.md)、[2026-05-27-undorewrite-consumer-alignment-design.md](2026-05-27-undorewrite-consumer-alignment-design.md)、[docs/guide/migration-undorewrite.md](../../guide/migration-undorewrite.md) |
| 与 gamestore 关系 | `gamestore-migrate` = 迁移路线图；`examples/gamestore` = 终态能力全集 |

## 1. 问题

`examples/gamestore` 展示接入**完成之后**的写法（代理写、`TxContext`、Rollback/Commit），README 明确不包含 `undorewrite` 执行步骤。集成方若从存量裸写出发，缺少仓库内可并排对照、可 `go test` 的**端到端切换参考**。

`docs/guide/migration-undorewrite.md` 与 `integration-checklist.md` 有命令说明，但无与 gamestore 同领域、步骤可跟跑的示例目录。

## 2. 目标

1. 新增 **`examples/gamestore-migrate/`** 独立 Go module（不 import 根包 `cow.Player`）。
2. 保留 **`before/`**（无 tag、无生成物、裸写 handler）与 **`after/`**（cow 模式金标准）两个 package，业务语义一致、便于 diff。
3. **A1 精简类型图**：单根 `Player`，覆盖标量、map、slice+append、指针嵌套写；省略 gamestore 的 `Guild` 第二根与 `Bags`/`Stats` 等复杂 Kind（README 链到 gamestore 补全）。
4. **C1 文档驱动**：README 列出人工执行的 `go generate` / `undorewrite` / `undocheck` 命令；`before/` 与 `after/` 均可 `go test`；CI **不**自动跑迁移脚本或 undorewrite golden。
5. 更新 `go.work`、CI、guide/gamestore README 互链。

## 3. 非目标

- 不在 CI 中从 `before/` 自动执行 undorewrite 并断言与 `after/` 字节一致（半自动 **C2** 不做）。
- 不复制 gamestore 全量类型图（双根、map map、map slice 等）。
- 不替代 `examples/gamestore` 的 `cmd/demo` 与全 FieldKind 演示。
- 不在示例 module 内 import `github.com/huangyuCN/cow` 业务类型（与 gamestore 一致，仅用 `replace` 跑工具）。

## 4. 方案选择

| 方案 | 结论 |
|------|------|
| A. 独立 module + `before/` / `after/` 双包对照 | **采用**（brainstorming 选定） |
| B. 仅在 gamestore 内增 `legacy/` 子目录 | 不采用（终态与迁移态混在同一包，易误导） |
| C. 仅 patch/README 片段、无长期双树 | 不采用（对照性差） |

交付形态：**方案 1 双包对照**；领域：**A1**；验证：**C1**。

## 5. 目录与模块

```text
examples/gamestore-migrate/
  go.mod
  doc.go
  README.md
  before/
    types.go
    handler.go
    handler_test.go
    fixture.go          # NewDemoPlayer 等（无 TxContext）
  after/
    types.go            # 同形 + // +cow:undoproxy-gen=true
    generate.go
    zz_generated.undo_proxy.go
    handler.go          # 代理写（与 undorewrite 预期输出一致）
    service.go          # txPool、runScopedCommit、runScopedWithRollback
    handler_test.go
    fixture.go
    demo.go             # 可选：fmt 打印 Commit/Rollback 结果
```

### 5.1 `go.mod`

```go
module github.com/huangyuCN/cow/examples/gamestore-migrate

go 1.25

require github.com/huangyuCN/cow v0.0.0

replace github.com/huangyuCN/cow => ../..
```

- `before` 与 `after` 为**同一 module 下两个 package**（`package beforeshop` / `package aftershop`，命名以避免与根包混淆，README 说明与 gamestore 领域对应关系）。
- `go.work` 增加 `./examples/gamestore-migrate`。

### 5.2 代码生成（仅 `after/`）

`after/generate.go`：

```go
//go:generate go run github.com/huangyuCN/cow/cmd/undoproxy-gen --output-file zz_generated.undo_proxy.go github.com/huangyuCN/cow/examples/gamestore-migrate/after
```

- `after/doc.go` 含 `// +cow:undoproxy-gen=package`（与 gamestore 约定一致）。
- 生成物**提交 Git**；改 `after/types.go` 后须 `go generate ./...`。

## 6. 类型图（A1 精简）

同包 struct；**单根** `Player`（`after/` 打 tag）。

### 6.1 `Player`

| 字段 | 类型 | 覆盖 Kind | before 裸写示例 |
|------|------|-----------|-----------------|
| `Gold` | `int64` | 标量 | `p.Gold = ...` |
| `Wallet` | `map[string]int64` | map 标量 | `p.Wallet["gold"] = ...` |
| `Items` | `[]*Item` | slice | `p.Items = append(...)` |
| `MainHero` | `*Hero` | 指针 struct | `p.MainHero.Level = ...` |

### 6.2 嵌套 struct（无 tag）

- `Item`：`Id int64`，`Name string`
- `Hero`：`Level int32`

`before/fixture.go` 提供 `NewDemoPlayer()`，保证 map/slice/指针非 nil，与 `after` 测试共用相同初始语义。

## 7. Handler 与测试对齐

### 7.1 `before/handler.go`

- 函数签名：`HandlePurchase(p *Player)`、`HandlePurchaseFail(p *Player) error`（**无** `*TxContext`）。
- 裸写逻辑与 `after` 中业务效果一致（扣 Gold、改 Wallet、append Items、升 MainHero.Level；Fail 路径改 Gold 后返回 error）。

### 7.2 `after/handler.go`

- 签名增加 `ctx *TxContext`。
- 使用生成代理：`PutGold`、`PutWallet`、`AppendItems`、`GetMainHeroForWrite` + `PutLevel` 等。
- 由维护者保证与对 `after/` 执行 `undorewrite -w` 后的结果一致（README 引导读者自行跑工具对照）。

### 7.3 `after/service.go`

与 gamestore 同语义：

- `runScopedCommit(fn func(ctx *TxContext) error) error`
- `runScopedWithRollback(fn func(ctx *TxContext) error) error`
- `txPool` 在 `zz_generated.undo_proxy.go` 或 `service.go` 中定义（与生成器产出一致即可）。

### 7.4 测试

| 包 | 测试 | 断言 |
|----|------|------|
| `before` | `TestHandlePurchase_*` | 裸写后字段值符合业务规则（无 Rollback） |
| `after` | `TestHandlePurchaseFail_Rollback` | Rollback 后 Player 与快照一致 |
| `after` | `TestHandlePurchaseSuccess_Commit` | Commit 后 Gold/Wallet 等符合预期 |
| `after` | `TestGenerated_contract`（可选） | 生成物含 `TxContext`、`Rollback`，且无 `AddUndo` |

`before` 测试**不**依赖 cow 生成物；`after` 测试**不**依赖根包 `cow` 的测试辅助函数。

## 8. README 迁移步骤（人工，C1）

主 README 固定 **8 步**（命令在 `examples/gamestore-migrate/README.md` 展开，此处为纲要）：

| 步 | 动作 | 说明 |
|----|------|------|
| 1 | 对照 | `diff -ru before/types.go after/types.go` 理解 tag |
| 2 | 打标 | 在业务包类型上添加 `// +cow:undoproxy-gen=true` |
| 3 | 生成 | 添加 `generate.go`，`go generate ./...` |
| 4 | 安装守门 | `go install ./cmd/undocheck`（此时裸写应触发 vet 诊断） |
| 5 | 改写 | 在**已 generate、handler 仍为裸写**的包上执行 `undorewrite ./...` dry-run → `-w`；结果应与 `after/handler.go` 一致 |
| 6 | 静态验收 | `go vet -vettool=.../undocheck ./...` |
| 7 | 运行时 | 对照 `after/service.go` 接入 `txPool` 与 `*TxContext` |
| 8 | 行为验收 | `go test ./...`；进阶见 `examples/gamestore` |

**练习方式（README 必写清）**：

- 推荐在**自有业务仓库**或本地副本按步骤操作；
- 本仓库 `after/` 为**金标准**；`before/` 为起点快照；
- 不要求读者改仓库内 `before/` 文件。
- 第 5 步练习态：将 `before/handler.go` 逻辑复制到已打标且已 `go generate` 的工作包（可本地建 `work/` 目录，**不**提交仓库）；仓库内 `after/handler.go` 即该步期望输出。

工具安装路径使用仓库根 `go install ./cmd/undorewrite` 等（与 [migration-undorewrite.md](../../guide/migration-undorewrite.md) 一致）。

## 9. CI 与 workspace

在 `.github/workflows/ci.yml` 增加 job（命名如 `example-gamestore-migrate`）：

```yaml
- run: cd examples/gamestore-migrate && go test ./before/... ./after/... -count=1
```

可选：`go run` / `go build` `after` 内 demo（若提供 `demo.go`）。

根 job `go test ./...` **不**包含 `examples/`（与 gamestore 相同策略）。

**不**在 CI 中执行：undorewrite、迁移 shell、before→after 自动同步。

## 10. 文档链接

| 文件 | 变更 |
|------|------|
| `examples/gamestore-migrate/README.md` | 新建：结构说明、8 步、命令、与 gamestore 对照表 |
| `examples/gamestore/README.md` | 增加「从裸写迁移」链到 migrate 示例 |
| `docs/guide/README.md` | 集成方索引增加 migrate 链接 |
| `docs/guide/integration-checklist.md` | 存量迁移节增加可运行示例链接 |
| 根 `README.md`（若存在 examples 索引） | 一行链到 migrate |

## 11. 验收标准

1. `cd examples/gamestore-migrate && go test ./before/... ./after/...` 成功。
2. `cd examples/gamestore-migrate/after && go generate ./...` 可再生成且与已提交 `zz_generated.undo_proxy.go` diff 为空（或仅允许注释差异）。
3. `after/` 无聚合根裸写；`go vet -vettool=undocheck ./after/...` 无 `cowbarewrite` 诊断。
4. `before/` 故意保留裸写且**无** tag；对 `before/` 跑 undocheck **不应**作为 CI 步骤（未监控则无诊断，README 说明仅在打标后出现守门）。
5. CI `example-gamestore-migrate` job 绿。
6. README 8 步与 `integration-checklist.md` 存量迁移项一致、无矛盾。

## 12. 与 gamestore 对照

| 维度 | gamestore-migrate | gamestore |
|------|-------------------|-----------|
| 根类型 | 单根 `Player` | `Player` + `Guild` |
| FieldKind | 4 类代表 | 全 Kind |
| 迁移步骤 | README 主交付 | 链接 migrate |
| undorewrite | README 引导在 after 包练习 | 仅文档链接 |
| demo | 可选 `after/demo.go` | `cmd/demo` |

## 13. 相关链接

- [2026-05-26-examples-gamestore-design.md](2026-05-26-examples-gamestore-design.md)
- [2026-05-27-undorewrite-consumer-alignment-design.md](2026-05-27-undorewrite-consumer-alignment-design.md)
- [migration-undorewrite.md](../../guide/migration-undorewrite.md)
- [integration-checklist.md](../../guide/integration-checklist.md)

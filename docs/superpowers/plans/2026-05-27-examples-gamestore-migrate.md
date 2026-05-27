# examples/gamestore-migrate 实现计划

> **状态：未开始**（截至 2026-05-27）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增独立 module `examples/gamestore-migrate/`，用 `before/`（裸写起点）与 `after/`（cow 模式金标准）双包对照，给用户提供从裸写到 `+cow`、`go generate`、`undorewrite`、`undocheck`、`TxContext` 运行时的完整切换参考。

**Architecture:** `examples/gamestore-migrate` 通过 `replace ../..` 引用仓库根以运行 `cmd/undoproxy-gen`、安装 `undorewrite/undocheck`。`before/` 不含 tag 与生成物，仅 `go test` 语义；`after/` 含 tag、生成物、`TxContext` + `txPool` + Commit/Rollback、代理写与回滚测试。CI 仅跑 `go test ./before/... ./after/...`（C1，不自动回放迁移脚本）。

**Tech Stack:** Go 1.25、`undoproxy-gen`、`undorewrite`、`undocheck`

**Spec:** `docs/superpowers/specs/2026-05-27-examples-gamestore-migrate-design.md`

---

## 文件结构（目标态）

| 文件 | 职责 |
|------|------|
| `examples/gamestore-migrate/go.mod` | 独立 module + `replace ../..` |
| `examples/gamestore-migrate/doc.go` | module 说明与包索引 |
| `examples/gamestore-migrate/README.md` | 迁移路线图（8 步）+ 对照说明 |
| `examples/gamestore-migrate/before/types.go` | 单根 `Player`（无 tag） |
| `examples/gamestore-migrate/before/fixture.go` | `NewDemoPlayer()` 等夹具（无 TxContext） |
| `examples/gamestore-migrate/before/handler.go` | 裸写 handler（Gold/Wallet/Items/MainHero） |
| `examples/gamestore-migrate/before/handler_test.go` | 纯语义断言（无 Rollback） |
| `examples/gamestore-migrate/after/doc.go` | `+cow:undoproxy-gen=package` |
| `examples/gamestore-migrate/after/types.go` | 同形 `Player` + `+cow:undoproxy-gen=true` |
| `examples/gamestore-migrate/after/generate.go` | `//go:generate ... undoproxy-gen` |
| `examples/gamestore-migrate/after/zz_generated.undo_proxy.go` | 生成物（提交） |
| `examples/gamestore-migrate/after/service.go` | `runScopedCommit` / `runScopedWithRollback` |
| `examples/gamestore-migrate/after/fixture.go` | 与 before 一致的初始化语义 |
| `examples/gamestore-migrate/after/handler.go` | 代理写（作为金标准） |
| `examples/gamestore-migrate/after/handler_test.go` | Rollback/Commit 断言 |
| `go.work` | 增加 `./examples/gamestore-migrate` |
| `.github/workflows/ci.yml` | 增加 `example-gamestore-migrate` job |
| `docs/guide/README.md` | 索引新增一行 |
| `examples/gamestore/README.md` | 增加迁移示例链接 |
| `docs/guide/integration-checklist.md` | “存量迁移”增加 migrate 链接 |

---

## Task 1: module 骨架

**Files:**
- Create: `examples/gamestore-migrate/go.mod`
- Create: `examples/gamestore-migrate/doc.go`

- [ ] **Step 1: 创建 `go.mod`**

```go
module github.com/huangyuCN/cow/examples/gamestore-migrate

go 1.25

require github.com/huangyuCN/cow v0.0.0

replace github.com/huangyuCN/cow => ../..
```

- [ ] **Step 2: 创建顶层 `doc.go`**

建议仅做索引说明，避免复制太多 guide 内容。

- [ ] **Step 3: 更新 `go.work`**

在 `use (...)` 中追加 `./examples/gamestore-migrate`。

---

## Task 2: `before/`（裸写起点，可测试）

**Files:**
- Create: `examples/gamestore-migrate/before/types.go`
- Create: `examples/gamestore-migrate/before/fixture.go`
- Create: `examples/gamestore-migrate/before/handler.go`
- Create: `examples/gamestore-migrate/before/handler_test.go`

- [ ] **Step 1: 定义类型图（无 tag）**

最小覆盖（与 spec A1 一致）：
- `Gold int64`
- `Wallet map[string]int64`
- `Items []*Item`
- `MainHero *Hero`

- [ ] **Step 2: 夹具 `NewDemoPlayer()`**

必须保证：
- `Wallet` 非 nil，含 `"gold"` 初始值
- `Items` 非 nil（可空切片）
- `MainHero` 非 nil，`Level` 有初始值

- [ ] **Step 3: handler（裸写）**

建议提供两个入口：
- `HandlePurchase(p *Player)`：成功路径（扣 Gold、改 Wallet、append Items、升 MainHero.Level）
- `HandlePurchaseFail(p *Player) error`：中途失败（先写入再返回 error，供 after 的 Rollback 对照）

- [ ] **Step 4: 测试**

只断言语义（无 Rollback）：
- 成功后字段按预期变化
- 失败函数返回 error 且字段已被修改（强调：这是“裸写时代”的风险）

---

## Task 3: `after/`（cow 模式金标准，可 generate、可 Rollback）

**Files:**
- Create: `examples/gamestore-migrate/after/doc.go`
- Create: `examples/gamestore-migrate/after/types.go`
- Create: `examples/gamestore-migrate/after/generate.go`
- Create: `examples/gamestore-migrate/after/service.go`
- Create: `examples/gamestore-migrate/after/fixture.go`
- Create: `examples/gamestore-migrate/after/handler.go`
- Create: `examples/gamestore-migrate/after/handler_test.go`
- Generate+Commit: `examples/gamestore-migrate/after/zz_generated.undo_proxy.go`

- [ ] **Step 1: `after/doc.go` 加包级 tag**

```go
// Package aftershop 演示从裸写迁移到 cow 写模式后的金标准代码。
//
// +cow:undoproxy-gen=package
package aftershop
```

- [ ] **Step 2: `after/types.go` 同形 + root tag**

对 `Player` 增加：
```go
// +cow:undoproxy-gen=true
```

其余字段与 before 保持一致，避免迁移路径引入“顺手改模型”的噪声。

- [ ] **Step 3: `generate.go`**

```go
//go:generate go run github.com/huangyuCN/cow/cmd/undoproxy-gen --output-file zz_generated.undo_proxy.go github.com/huangyuCN/cow/examples/gamestore-migrate/after
```

- [ ] **Step 4: `go generate ./...` 生成并提交 `zz_generated.undo_proxy.go`**

验收：
```bash
cd examples/gamestore-migrate/after
go generate ./...
```

- [ ] **Step 5: `service.go`（TxContext 作用域）**

与 `examples/gamestore/service.go` 同语义：
- `runScopedWithRollback(fn func(ctx *TxContext) error) error`
- `runScopedCommit(fn func(ctx *TxContext) error) error`

- [ ] **Step 6: `after/handler.go`（代理写）**

签名必须包含 `ctx *TxContext`，并使用生成 API：
- `p.PutGold(ctx, ...)`
- `p.PutWallet(ctx, \"gold\", ...)`
- `p.AppendItems(ctx, ...)`
- `p.GetMainHeroForWrite(ctx).PutLevel(ctx, ...)`

---

## Task 4: README（8 步路线图，C1）

**Files:**
- Create: `examples/gamestore-migrate/README.md`

- [ ] **Step 1: 写清“如何使用本示例”**

必须包含三层说明（避免误解）：
- `before/` 是起点快照（裸写）
- `after/` 是金标准（迁移完成态）
- 第 5 步（undorewrite）建议读者在本地临时工作包练习（不要改仓库内 `before/`）

- [ ] **Step 2: 8 步命令（可复制粘贴）**

必须覆盖：
- `go generate`
- `go install ./cmd/undorewrite` + `undorewrite` dry-run/`-w`
- `go install ./cmd/undocheck` + `go vet -vettool=...`
- `go test`

- [ ] **Step 3: 与 `examples/gamestore` 的关系**

给出“进阶学习”的链接与差异表（单根 vs 双根、Kind 覆盖范围等）。

---

## Task 5: CI 与文档联动

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `docs/guide/README.md`
- Modify: `examples/gamestore/README.md`
- Modify: `docs/guide/integration-checklist.md`

- [ ] **Step 1: CI 增加 job**

建议与 `example-gamestore` 并列：
```bash
cd examples/gamestore-migrate && go test ./before/... ./after/... -count=1
```

- [ ] **Step 2: guide 索引增加一行链接**
- [ ] **Step 3: gamestore README 增加“迁移示例”链接**
- [ ] **Step 4: integration checklist 增加 migrate 链接**

---

## 验收清单

- [ ] `cd examples/gamestore-migrate && go test ./before/... ./after/... -count=1` 全绿
- [ ] `cd examples/gamestore-migrate/after && go generate ./...` 可重复执行且无变化（或仅注释差异）
- [ ] `cd examples/gamestore-migrate/after && go vet -vettool=$(go env GOPATH)/bin/undocheck ./...` 无 `cowbarewrite` 诊断
- [ ] `.github/workflows/ci.yml` 新 job 通过


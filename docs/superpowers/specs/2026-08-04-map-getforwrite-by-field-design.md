# map[k]*Struct GetForWrite 按字段名命名设计说明

| 项 | 值 |
|---|---|
| 状态 | **已实现** |
| 日期 | 2026-08-04 |
| 模块 | `internal/cowgen`、`cmd/undoproxy-gen`、`internal/cowproxy`、`cmd/undorewrite` |
| 优先级 | P0（同包多 map 字段共享元素类型时生成物无法编译） |
| 前置 | [2026-08-04-undoproxy-gen-scalable-codegen-design.md](2026-08-04-undoproxy-gen-scalable-codegen-design.md) |
| 现行文档 | [docs/guide/proxy-api.md](../../guide/proxy-api.md)、[docs/guide/codegen-undoproxy.md](../../guide/codegen-undoproxy.md) |
| 取代 | 原 `docs/prd/2026-08-04-map-getforwrite-by-field.md`（已删除，内容并入本文） |

## 1. 问题

对 `map[K]*T`，生成器与 catalog 用**元素类型名**拼 GetForWrite：

```go
// 现状（有问题）
MapKeyGetForWriteName(plan.ElemName) // → GetStackableItemForWrite
// emitStructuredMapPtrGet 直接用 plan.ElemName 进 Get%sForWrite
```

同 struct 内多个字段共享元素类型时重定义。宿主实锤：

```go
type Bag struct {
    Stackable        map[uint64]*StackableItem
    VirtualStackable map[uint64]*StackableItem // 再次生成 GetStackableItemForWrite
    Unique           map[uint64]*UniqueItem
    VirtualUnique    map[uint64]*UniqueItem
}
```

`Put{Field}` / `Remove{Field}` / `Get{Field}MapForWrite` / 指针 `Get{Field}ForWrite` 已按**字段名**；惟独 map 元素 GetForWrite 走 `ElemName`，不一致。

## 2. 目标

1. **单层** `map[K]*Struct` 与 **双层** `map[K]map[K2]*Struct` 的元素 GetForWrite 均命名为 **`Get{Field}ForWrite`**，同接收者上唯一。
2. `emit_structured`（`emitStructuredMapPtrGet` / `emitStructuredMapMapPtrGet`）、`cowproxy` catalog、`undorewrite` 共用 `MapKeyGetForWriteName(field)`。
3. cow 仓硬切：测试、examples、guide 中旧调用一律改为新名；**不**生成旧名别名。
4. 夹具证明：单 struct 两个同元素类型的 map（单层或双层）字段可生成且可编译。

## 3. 非目标

| 不做 | 说明 |
|------|------|
| 改 `ElemAtForWriteName`（`Get{Elem}AtForWrite`） | 仍按元素类型；撞名风险另开 follow-up |
| 宿主手写 API / `Remove*` 前缀字段与生成方法冲突 | 宿主改名或另开设计 |
| undo 语义 / `undoKind` | 仅方法名 |
| 过渡期双名别名 | 硬切，见 §4 |
| 强制合并 `PtrGetForWriteName` 与 `MapKeyGetForWriteName` | 二者输出形态相同；YAGNI 可保持两函数 |

## 4. 方案选择

| 方案 | 结论 |
|------|------|
| **A. `MapKeyGetForWriteName(field)` → `Get{Field}ForWrite`，硬切** | **采用** |
| B. 仅当同 Elem 多字段时按字段名，否则保留 Elem 名 | 拒绝（规则分叉、catalog 难文档化） |
| C. 生成旧 `Get{Elem}ForWrite` 别名作过渡 | 拒绝（双轨污染；集成方仍须迁一次） |
| D. 本设计一并改 `ElemAtForWriteName` | 不做（爆破面更大；无当前宿主阻塞证据） |

## 5. 命名与接线

### 5.1 规则（钉死）

```go
// MapKeyGetForWriteName 按字段名生成 map[k]*Struct 的 GetForWrite。
func MapKeyGetForWriteName(field string) string {
    return "Get" + field + "ForWrite"
}
```

入参是**字段名**，禁止再传入 `ElemName` 或 `Singular(field)`。

| 字段 | 新方法名 |
|------|----------|
| `Stackable` | `GetStackableForWrite` |
| `VirtualStackable` | `GetVirtualStackableForWrite` |
| `Heros` | `GetHerosForWrite`（**不再** `GetHeroForWrite`） |
| `Skills` | `GetSkillsForWrite` |

### 5.2 破坏性对照（常见旧调用）

| 旧（Elem/Singular） | 新（Field） |
|---------------------|-------------|
| `GetHeroForWrite`（字段 `Heros`） | `GetHerosForWrite` |
| `GetSkillForWrite`（字段 `Skills`） | `GetSkillsForWrite` |
| `GetStackableItemForWrite`（字段 `Stackable`） | `GetStackableForWrite` |

指针字段 `GetMainHeroForWrite` 等本就按字段名，**不变**。

### 5.3 改动位点

| 文件 | 动作 |
|------|------|
| `internal/cowgen/naming.go` | 改 `MapKeyGetForWriteName` 语义与注释；补/改单测 |
| `cmd/undoproxy-gen/emit_structured.go` | `emitStructuredMapPtrGet` / `emitStructuredMapMapPtrGet`：方法名用 `MapKeyGetForWriteName(plan.FieldName)` |
| `internal/cowproxy/catalog.go` | `KindMapPtrStruct` 与 `KindMapMapPtrStruct` 均 `MapKeyGetForWriteName(plan.FieldName)` |
| 断言 `GetHeroForWrite` 等的测试 / examples / handler | 改为 `GetHerosForWrite` 等 |
| `docs/guide/proxy-api.md` 等 | 示例同步；注明破坏性变更 |
| 生成物 `zz_generated.undo_proxy.go`（根包与 examples） | `go generate` 再生 |

`undorewrite` 经 catalog 取名，无需单独硬编码旧逻辑；其测试期望字符串随 catalog 更新。

## 6. 测试与验收

### 6.1 Must 测试（TDD）

1. **命名**：`MapKeyGetForWriteName("VirtualStackable") == "GetVirtualStackableForWrite"`。
2. **撞名夹具**：testdata（或等价）中单根含 `Stackable` / `VirtualStackable` 两个 `map[K]*SameElem` → 生成源码含 `GetStackableForWrite` 与 `GetVirtualStackableForWrite` 各一次定义，且**不含**重复的 `GetStackableItemForWrite`（若 Elem 名为 StackableItem）。
3. **catalog**：对 `KindMapPtrStruct`，`GetForWrite == "Get"+Field+"ForWrite"`。
4. **回归**：凡期望旧 map Get 名处改为新名；`go test` 相关包通过。

### 6.2 cow 验收

```bash
go test ./cmd/undoproxy-gen ./internal/cowgen ./internal/cowproxy ./cmd/undorewrite -count=1
# 根包
go generate .
# examples 为嵌套 module，勿在仓库根依赖 ./examples/...
(cd examples/gamestore && go generate . && go test ./... -count=1)
(cd examples/gamestore-migrate/after && go generate . && go test ./... -count=1)
```

验收清单：

- [ ] 上述命令全绿
- [ ] 双 map 同 Elem 夹具通过
- [ ] guide 已写新命名与破坏性说明

### 6.3 非 Must

宿主（如 shimmer/server）`go generate` 后不再出现 `GetStackableItemForWrite already declared`；调用点需按新名手迁或再跑 `undorewrite`。实现者可自证并写入 PR 描述。

## 7. 风险

| 风险 | 缓解 |
|------|------|
| 已接入项目编译失败 | changelog/guide 列旧→新；硬切一次说清 |
| 漏改某处旧字符串断言 | `rg 'GetHeroForWrite\|GetSkillForWrite'` 全仓扫 |
| `ElemAtForWrite` 仍可能撞名 | 记 follow-up，本设计不扩 scope |

## 8. Follow-up（非本交付）

- `ElemAtForWriteName`：改为按字段（或 `Get{Field}AtForWrite`）以防 `map[k][]*Same` 多字段撞名。
- 宿主 `Remove*` 前缀字段 / 手写方法与生成 API 冲突。

## 9. 建议实现顺序

1. TDD：命名单测 + 双 map 夹具（RED）。
2. 改 `MapKeyGetForWriteName` + emit + catalog → GREEN。
3. 全仓替换旧调用期望；regenerate；改 guide。
4. `undorewrite` / examples 测试全绿。

建议提交说明（经确认后再 commit）：

```text
fix(undoproxy-gen): name map GetForWrite after field, not elem type

Avoid duplicate GetStackableItemForWrite when Bag has both Stackable
and VirtualStackable maps of *StackableItem.
```

## 10. 成功判据

> 同一接收者上任意多个 `map[K]*T` 字段各自拥有唯一的 `Get{Field}ForWrite`；与 `Put{Field}` / `Remove{Field}` 命名轴一致；cow 相关测试与 examples 全绿。

# Mega Player 夹具与 Benchmark 设计说明

| 项 | 值 |
|---|---|
| 状态 | 已实现（2026-05-25） |
| 模块 | `github.com/huangyuCN/cow` |
| 需求来源 | 用户：构造 ~1MB 游戏向 `Player`，验证 `undoproxy-gen` 完整性，对比全量 `DeepCopy` |
| 前置 | [undoproxy-codegen-design.md](./2026-05-25-undoproxy-codegen-design.md)、[cow-undo-log-mvp-design.md](./2026-05-25-cow-undo-log-mvp-design.md) |

## 1. 目标

1. **模型**：扩展 `Player` 及同包子 struct，覆盖游戏服常见嵌套（`map` / `slice` / 指针 / `map[k]map` / `map[k][]`），堆上常驻图 **约 1MB**（±15%）。
2. **正确性**：在 mega 夹具上验证 `undoproxy-gen` 生成代理的**全覆盖探针**与**业务短路径**均可 `Rollback` 无损。
3. **性能**：新增 mega 档 Benchmark，与 `deepcopy-gen` 全量 `Player.DeepCopy()` + 稀疏写对照；结果可归档至 `docs/superpowers/benchmarks/`。

## 2. 非目标

- BSON/JSON 序列化体积达标（本期仅堆估算；见 §4）。
- 并发 / 多 goroutine 压测。
- 修改 `undoproxy-gen` 语义（除非 mega 暴露生成器 bug）。
- `MegaPlayer` 第二套根类型（采用**单 `Player` + 双档构造**）。

## 3. 已确认决策（brainstorming 2026-05-25）

| 决策项 | 选择 |
|---|---|
| 体积口径 | **A**：堆上对象图约 **1MB**，允许 **±15%** |
| 夹具策略 | **A**：类型一套；`newBenchPlayer()` lite + `newMegaBenchPlayer()` mega |
| 测试策略 | **C**：Benchmark **短路径**；单测 **全覆盖探针** + 业务路径回滚 |

## 4. 体积验收

### 4.1 估算函数

实现 `approxPlayerHeapBytes(p *Player) uint64`（`bench_mega_fixture.go`）：

- 遍历可达指针、slice、map：累加 `string` 的 `len`、slice 长度 × 元素估算大小、map 条目数 × 平均价值。
- 不调用 `unsafe.Sizeof` 对整个图做精确扫描；允许 ±15% 误差。
- **不**把 map 桶内部实现细节算进公式，用经验系数即可。

### 4.2 断言

```go
func TestMegaFixtureSize(t *testing.T) {
    p := newMegaBenchPlayer()
    got := approxPlayerHeapBytes(p)
    const (
        want   = 1 << 20 // 1 MiB
        margin = 15       // percent
    )
    lo := want * (100 - margin) / 100
    hi := want * (100 + margin) / 100
    if got < lo || got > hi {
        t.Fatalf("heap approx %d want [%d,%d]", got, lo, hi)
    }
}
```

调参入口：`bench_mega_fixture.go` 顶部 **规模常量**（见 §5.2），调一次跑 `TestMegaFixtureSize` 直至落入区间。

## 5. 数据模型

### 5.1 类型布局

`Player` 保留现有 lite 字段（`Assets`、`Items`、`Hero`），**新增** mega 字段（均需 `undoproxy-gen` 可达类型）：

| 字段 | 类型 | 生成器分类 | 作用 |
|------|------|------------|------|
| `Uid` 等 | 标量 | Scalar | 基线 |
| `Assets` | `map[string]int64` | MapScalar | lite + mega |
| `Items` | `[]*Item` | SlicePtr | Remove/Truncate/Append |
| `Hero` | `*Hero` | PtrStruct | lite 单英雄 |
| `Heros` | `map[int32]*Hero` | MapPtrStruct | 多英雄 + `GetHeroForWrite` |
| `Bags` | `map[int32][]*Item` | MapSlicePtr | 背包分页 |
| `Stats` | `map[int32]map[string]int64` | MapMapScalar | 嵌套 map |
| `Cooldowns` | `map[int32][]int32` | MapSliceValue | 冷却列表 |
| `Mails` | `map[uint64]*Mail` | MapPtrStruct | 大 `string` 拉高体积 |
| `Quests` | `map[int32]*Quest` | MapPtrStruct | 任务进度 |

子类型（`types_game.go`，同包、`+k8s:deepcopy-gen=true`）：

- `Item`：已有，扩展可选字段 `Extra string`（可选，用于体积）
- `Hero`：`HeroId`、`Level`、`Skills map[int32]*Skill`
- `Skill`：`SkillId`、`Level`
- `Mail`：`Id`、`Subject`、`Body string`（**Body 长度主导 mail 体积**）
- `Quest`：`Id`、`State`、`Objectives map[int32]int32`

`Player` 保持唯一 `// +cow:undoproxy-gen=true` 根；子 struct 由可达性纳入生成图。

### 5.2 初版规模常量（可调）

| 常量 | 初值 | 说明 |
|------|------|------|
| `megaItemCount` | 2000 | `Items` 条数 |
| `megaItemNameLen` | 48 | 物品名长度 |
| `megaHeroCount` | 120 | `Heros` 数量 |
| `megaSkillsPerHero` | 40 | 每英雄技能数 |
| `megaBagCount` | 30 | 背包页数 |
| `megaItemsPerBag` | 40 | 每页物品数 |
| `megaMailCount` | 80 | 邮件数 |
| `megaMailBodyLen` | 4096 | 邮件正文（体积主力） |
| `megaQuestCount` | 100 | 任务数 |
| `megaStatGroups` | 12 | `Stats` 外层 key 数 |
| `megaStatKeysPerGroup` | 50 | 内层 string key 数 |
| `megaCooldownKeys` | 80 | 冷却 map 大小 |
| `megaCooldownListLen` | 24 | 每个 cooldown slice 长度 |
| `megaAssetCount` | 200 | `Assets` 条目 |

实现时按 `TestMegaFixtureSize` 微调，优先动 `megaMailBodyLen`、`megaItemCount`、`megaHeroCount*megaSkillsPerHero`。

### 5.3 Lite 构造（不变语义）

`newBenchPlayer()`：**仅**填充 `Uid`、`Assets`（~100）、`Items`（~500）、`Hero`；mega 字段保持 `nil`。保证现有 MVP 测试与 B 档 Benchmark 耗时不变。

## 6. 正确性测试

### 6.1 全覆盖探针 `TestMegaPlayer_ProxyProbe_Rollback`

对 `newMegaBenchPlayer()` 快照 `want := clonePlayerSnapshot(p)`，在 `runScopedWithRollback` 内依次执行（顺序固定，便于回归）：

| # | 代理类型 | 示例调用（以生成为准） |
|---|----------|------------------------|
| 1 | Scalar | `PutUid` / `PutLevel` |
| 2 | MapScalar | `PutAssets` |
| 3 | Map Put 指针 | `PutHeros` 或 `GetHeroForWrite` + 改 `PutSkills` |
| 4 | Map Get 写 | `GetHeroForWrite` → 改 `Level` |
| 5 | Slice Append | `AppendItems` |
| 6 | Slice Set | `SetItemsAt` |
| 7 | Slice Remove | `RemoveItemsAt` |
| 8 | Slice Truncate | `TruncateItems` |
| 9 | MapSlice Append | `AppendBagsAt` |
| 10 | MapSlice Remove | `RemoveBagsAt` |
| 11 | MapMap Put | `PutStats(ctx, group, key, val)` |
| 12 | MapMap GetMap | `GetStatsMapForWrite` + 内层写 |
| 13 | MapSliceValue | `AppendCooldownsAt` 或 `SetCooldownsAt` |

结束后 `errors.New("rollback")`；`assertPlayerEqual(t, p, want)`。

探针序列**故意**触及嵌套与删截断；若某字段未生成对应方法，编译期即失败。

### 6.2 业务短路径 `TestMegaPlayer_BusinessPath_Rollback`

与 §7 Benchmark 相同的 6–8 步（`applyMegaSparseWrites`），验证回滚一致。

### 6.3 与 `cmp` 选项

扩展 `assertPlayerEqual` 的 `cmp.Comparer`，覆盖 `Mail`、`Quest`、`Skill` 等指针类型（按 Id/关键字段比较，避免比指针地址）。

## 7. Benchmark

### 7.1 短路径 `applyMegaSparseWrites`

固定步骤（示例，实现时与生成方法名对齐）：

1. `PutAssets(ctx, "gold", newVal)`
2. `h := GetHeroForWrite(ctx, knownHeroId)`；`h.PutLevel(ctx, 99)`
3. `AppendItems(ctx, newItem)`
4. `PutHeros(ctx, heroId, skillPtr)` 或 `GetSkillForWrite` 改技能
5. `AppendBagsAt(ctx, bagId, item)`
6. `PutStats(ctx, statGroup, statKey, value)`
7. `RemoveItemsAt(ctx, 0)` 或 Truncate（二选一，保持可重复回滚）

Direct 对照：`sparseWriteMegaDirect` 在 `DeepCopy` 副本上改对应字段（无 Undo）。

### 7.2 Benchmark 表

| 名称 | 夹具 | 循环内行为 |
|------|------|------------|
| `BenchmarkMega_UndoLog_SparseWrite_Rollback` | mega | Pool → `applyMegaSparseWrites` → `Rollback` |
| `BenchmarkMega_UndoLog_SparseWrite_Commit` | mega | Pool → 写 → `Reset`；**禁止**计时外 `DeepCopy` |
| `BenchmarkMega_DeepCopyGen_SparseWrite` | mega | `work := seed.DeepCopy()` → `sparseWriteMegaDirect(work)` |

保留原 `BenchmarkUndoLog_*` / `BenchmarkDeepCopyGen_*`（lite），便于历史对比。

### 7.3 归档

跑完后：

```bash
go test -bench='BenchmarkMega_|BenchmarkUndoLog|BenchmarkDeepCopyGen' -benchmem -count=5 ./...
```

用 `benchstat` 对比 lite 与 mega、Undo vs DeepCopy，追加章节至  
`docs/superpowers/benchmarks/cow-mega-player-benchmark.md`（含日期、go version、commit、命令、表格）。

## 8. 文件与集成

| 文件 | 操作 |
|------|------|
| `types.go` | 扩展 `Player` 字段 |
| `types_game.go` | 新建子类型 |
| `bench_mega_fixture.go` | `newMegaBenchPlayer`、`approxPlayerHeapBytes`、常量 |
| `bench_fixture_test.go` | lite 构造保持/收紧 |
| `player_mega_test.go` | 探针 + 业务路径 |
| `benchmark_mega_test.go` | mega 三组 bench |
| `zz_generated.undo_proxy.go` | `go generate` 后更新 |
| `zz_generated.deepcopy.go` | 类型变更后 `go generate` |

流程：

1. 扩展类型 → `go generate ./...`
2. 实现 mega 构造，跑 `TestMegaFixtureSize` 调参
3. 实现探针与 `applyMegaSparseWrites`
4. 跑全量 `go test ./...`
5. 跑 Benchmark，询问是否归档

## 9. 验收标准

- [ ] `TestMegaFixtureSize`：堆估算 ∈ [0.85, 1.15] MiB
- [x] `TestMegaPlayer_ProxyProbe_Rollback` / `_Commit` 通过
- [x] `TestMegaPlayer_BusinessPath_Rollback` / `_Commit` 通过
- [x] `TestMegaPlayer_CommitPersistsAfterLaterRollback` 通过
- [ ] 现有 lite 测试与 B 档 Benchmark 仍通过
- [ ] `BenchmarkMega_UndoLog_Rollback` 的 `ns/op`、`allocs/op` 显著优于 `BenchmarkMega_DeepCopyGen_SparseWrite`
- [ ] mega 结果已记录或待用户确认归档

## 10. 参考

- [2026-05-25-undoproxy-codegen-design.md](./2026-05-25-undoproxy-codegen-design.md)
- [cow-undo-log-mvp-benchmark.md](../benchmarks/cow-undo-log-mvp-benchmark.md)
- `new.md` — 性能目标背景

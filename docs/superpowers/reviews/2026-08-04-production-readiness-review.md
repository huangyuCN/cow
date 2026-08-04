# cow 生产可用性代码审计报告

- **日期**：2026-08-04
- **审计对象**：HEAD `8241174`（工作区干净；评审后发起者修复了一处 `go vet` 失败，见「已现场修复」，该修复**尚未提交**）
- **审计范围**：以当前工作区代码为主要审计对象；git diff 范围 `19129ff..8241174`（15 个提交，v2 runtime 全量生成与结构化 Undo 收敛期）用于理解近期演进
- **审计方式**：资深评审子代理逐文件审读 + 实跑 `go build ./...`、`go vet ./...`、`go test ./...`（主仓全绿）、examples 两个独立 module 测试（全绿）、`wc -l` 全量核对文件规模
- **未逐行覆盖**：`cmd/undocheck`、`cmd/undorewrite`、`internal/cowproxy` 正文（测试通过、行数合规、testdata 覆盖 homonym/importscope/consumer 场景，属间接验证）

## 结论

**是否达到生产可用：修复后可用。**（2026-08-04 更新：Important #1–#4 与全部 Minor 已修复，见文末「修复记录」，均未提交。）

架构收敛干净、生成路径统一、零分配 typed slot 与池化 reset 设计正确且全部测试通过，工程完成度高；但生成代码存在 `had`/`had2` 回滚标志错位的真实缺陷（当前良性但契约已损坏），且 `CloneForWrite` fork 场景下 `RemoveAt` 会对保留的原根造成持久数据损坏——这两点必须先修复或以文档+测试显式排除，才能承担生产责任。

## 优点

1. **双轨收敛干净，无死代码残留**。`tx.go`、`zz_generated.undo_proxy_v2.go`、`tx_v2_*` 三组测试已整体删除，只有一条 `cowmon.LoadPackage → cowgen.BuildGraph → emitFromGraph` 路径（cmd/undoproxy-gen/main.go:26-36，emit.go:7-9 仅一层转发）。生成物中无 V2/AddUndo 痕迹。
2. **typed undoOp 槽位设计高效**（zz_generated.undo_proxy.go:72-96）：标量旧值、key、slice 快照、内层 map 均为具名字段，写入路径零装箱、零额外分配，`push` 只是一次 struct 值拷贝；`uint16 undoKind` + `checkUndoKindCount`（cmd/undoproxy-gen/emit_undo.go:252-257）显式守住 65535 上限，大类型图会报明确错误而非静默溢出。
3. **池化 TxContext 的引用清理有测试兜底**：`Reset()` 逐槽清零（生成物 110-115 行），`tx_reset_test.go:33-45` 直接断言底层数组槽位不再持有 player/hero/字符串/snap 引用。`runScopedWithRollback/Commit`（examples/gamestore/service.go:4-25）Get→Reset→使用→(Rollback)→Put 的生命周期正确。
4. **身份匹配修复方向正确**：`MonitoredSet` 以 `*types.TypeName` 对象身份为主键、路径+名称为跨加载回退（internal/cowmon/reachable.go:28-47），testdata 有 `homonym` 用例，与提交 `74c8249` 的意图一致。
5. **跨包叶子类型处理完备**：`Qualifiers.Alias` 有同 path 幂等与冲突递增逻辑（internal/cowgen/qualifiers.go:24-43），`writeImports` 与 TypeStr 共用同一张别名表（cmd/undoproxy-gen/emit_structured_graph.go:37-49），classify 对跨包 named basic 走 TypeStr 保留限定名（internal/cowgen/classify.go:89-100）。四份 2026-08-04 设计的产物特征在生成物中全部可见。
6. **生成器测试结构好**：按关注点拆分（emit_undo_slots_test、emit_undo_scalar_test、emit_map_getforwrite_test、emit_imports_test、emit_clone_dedupe_test、generate_golden_test），golden 测试覆盖生成物回归。
7. **规范总体达标**：中文注释、导出名不以包名开头均落实；手写文件仅一例超 500 行（见 Important #3）。

## 问题

### Critical

本轮未发现无争议的 Critical 级缺陷。Important #1 若文档承诺「CloneForWrite 可保留原快照并行使用」，则升级为 Critical（数据损坏）。

### Important

1. **fork 语义下根级 slice 字段存在持久损坏风险**
   - 位置：生成物 `RemoveItemsAt`（zz_generated.undo_proxy.go:492-497）、模板 cmd/undoproxy-gen/emit_structured.go:168-173；同类 `RemoveBagsAt`（zz_generated 586-595 行）因结果写回共享 map 槽位而不受影响。
   - 推理：`CloneForWrite` 浅拷贝 slice 头（zz_generated.undo_proxy.go:427-444），session 与原根共用底层数组。`p.Items = append(p.Items[:i], p.Items[i+1:]...)` 在共享底层数组上**原地左移**；Rollback 时 session 得到 `append(nil, snap...)` 的新副本，**原根仍指向被移位过的旧数组**（元素丢失 + 尾元素重复），损坏在回滚后永久存在。现有测试全部直接写原对象，从未覆盖「保留原根 + fork + 回滚」路径。
   - 为何重要：`CloneForWrite`（可写浅拷贝）是导出 API，fork 用法是合理预期。
   - 修法（二选一）：a) RemoveAt 改为写时复制——已有 `tail` 快照副本，直接 `p.Items = append(tail[:i], tail[i+1:]...)`，每次删除多一次分配，换 fork 安全；b) 在 limitations 文档明确禁止双根并存并加防御性测试断言该约定。**建议 a**。

2. **`had2` 标志只写不读，Rollback 分支用错标志**（生成器缺陷，当前后果良性）
   - 位置：cmd/undoproxy-gen/emit_structured.go:324、:357 push 侧写 `had2:`，而 kind `MapMapOuterRestore` 的回滚体（:345-347）判断 `op.had`。生成物体现为 zz_generated.undo_proxy.go:661 vs 223-228。
   - 影响：外层 key 原本「存在但值为 nil map」时，回滚会 delete 该 key 而非恢复 nil 槽；因所有生成 API 将 `!ok || inner==nil` 等价处理，当前无行为差异。但这是回滚逻辑与记录逻辑的契约错位，新增 kind 时极易踩雷，且 `had2` 成为死字段。
   - 修法：统一改用 `had`，或在回滚体中真正消费 `had2` 恢复 nil 槽；顺带清理语义并补 nil-inner 回滚测试。

3. **cmd/undoproxy-gen/emit_structured.go 530 行，违反 AGENTS.md ≤500 行约束**
   - 修法：将 map-of-slice / map-of-map 两组发射器拆到 `emit_structured_mapmap.go`（现有 emit_structured_write_ext.go 的拆分方式可作先例）。

4. **examples 不在根 module，`go test ./...` 覆盖不到**
   - 若 CI 只跑根目录会漏测两个示例 module。修法：CI 增加 `cd examples/* && go test ./...`，或文档明示。

5. **审计盲区声明**：cmd/undocheck、cmd/undorewrite、internal/cowproxy 正文本轮未逐行审读。生产发布前建议补一轮针对性评审。

### Minor

1. 手写冒泡排序出现在两处：cmd/undoproxy-gen/emit_undo.go:364-377 `sortedMapKeys`、internal/cowmon/reachable.go:105-111；同级代码 qualifiers.go 已用 `sort.Strings`，风格不一，建议统一用标准库。
2. internal/cowmon/reachable.go:94-96 的外部类型报错分支不可达（`structRefsInType` 已在 :130 过滤为同包），且跨包 struct 会被 walk 静默递归，实际报错来自 classify。建议让 walk 对跨包 struct 直接报错并删除死分支。
3. `scalarOldField("")` 回退返回未注册的 `oldI64`（cmd/undoproxy-gen/emit_undo.go:173-175），是潜在脚枪，建议改为显式报错。
4. 生成方法不做下标/空 map 防御（如 SetItemsAt、MapSliceElemSet），panic 责任全在调用方；既然是对外 API，建议在 proxy-api 文档中集中声明该契约。
5. `Singular` 仅去尾 s（internal/cowgen/naming.go:4-9），「Address→Addres」这类会产出怪方法名；可加简单豁免表或在文档注明。
6. undoOp 为每个可达 struct 携带一个指针槽、为每类 key/scalar/snap 携带一个槽，大类型图下 struct 会线性膨胀（本仓约 150+ 字节/op），每次 push/Reset 都是全量拷贝/清零。当前规模无碍，建议在 limitations 文档记录该扩展性特征。

## 建议（修复顺序）

1. 先修 Important #2（标志错位）与 #1（fork 损坏，推荐写时复制方案），各配一个回归测试：fork+RemoveAt+Rollback 断言原根不变；nil-inner 外层回滚断言槽位恢复。
2. 拆分 emit_structured.go 至 ≤500 行；顺手统一排序工具函数、清理死分支。
3. 在 docs/guide/limitations.md 固化三条契约：单活根/单协程模型、生成 API 不做边界检查、undo 日志内存与类型图规模的关系。
4. CI 增加：examples 两 module 测试、`go vet ./...` 门禁、（若可行）用 undocheck 自扫仓库示例。
5. 安排对 cmd/undocheck、cmd/undorewrite、internal/cowproxy 的专项评审后再对外发布。

## 已现场修复（评审过程中，未提交）

- `bench_baseline_fixture.go:37`：`go vet` 报 `append with no values`，`append(p.Items[:len(p.Items)-1])` 已改为等价截断 `p.Items[:len(p.Items)-1]`。修复后 `go vet ./...` 与 `go test ./...`（主仓 + examples 两 module）全绿。

## 修复记录（2026-08-04，均未提交）

全部按 TDD（RED→GREEN）完成，主仓与 examples 两 module 的 build/vet/test 全绿。

| 问题 | 修复 | 回归测试 |
|------|------|----------|
| Important #1 fork 损坏 | 根级 slice `RemoveAt` 模板改为写时复制：快照与结果各自独立数组（`make` 精确容量，避免中途再分配），不再移位共享底层数组；map-slice 变体经核实由 map 槽快照恢复、不受影响 | `undo_semantics_test.go` `TestRollback_ForkRemoveAtKeepsOriginalRoot` |
| Important #2 had/had2 错位 | 四处外层 push 统一改写 `had`（key 原存在性），三个 `*OuterDelete` kind 改为 `*OuterRestore` 恢复体（按 `had` 恢复 nil 槽或删除 key），删除 `had2` 字段；map-of-map 两 kind 去重为一 | `undo_semantics_test.go` `TestRollback_MapMapNilInnerSlotRestored` |
| Important #3 530 行超限 | map-of-map 段拆出 `cmd/undoproxy-gen/emit_structured_mapmap.go`（242 行），主文件 306 行；生成物逐字节不变 | 既有断言测试守护 |
| Important #4 examples 漏测 | CI 已有独立 job 覆盖（审计遗漏），另补 `go vet ./...` 门禁入 test job（`.github/workflows/ci.yml`） | — |
| Minor #1/#2 冒泡排序 | `sortedMapKeys` 用 `sort.Strings`；`CollectReachable` 排序用 `slices.SortFunc` | 既有测试 |
| Minor #2 死分支 | `structRefsInType` 对跨包命名 struct 显式报错并点名类型；删除 `CollectReachable` 不可达外检分支，错误经 `%w` 携带字段上下文 | `reachable_external_test.go` |
| Minor #3 scalarOldField | 空类型串由静默回退 `oldI64` 改为 panic 显式失败 | `TestUndoBuilder_ScalarOldField_emptyPanics` |
| Minor #4/#6 契约文档 | `docs/guide/proxy-api.md` 增 fork 语义与「无边界检查」契约；`docs/guide/limitations.md` 增单活根、无边界检查、undoOp 内存与类型图规模三条 | — |

**性能影响（benchstat，基线 8241174 vs 修复后，Apple M3，count=5）**：COW 使 `RemoveItemsAt`（mega 档 2000 元素）每次多一次全量快照拷贝：

| Benchmark | ns/op | B/op | allocs/op |
|-----------|------:|-----:|----------:|
| `Mega_UndoLog_SparseWrite32_Rollback` | **+21.9%**（6.62→8.06µs） | **+45.7%**（35.05→51.05KiB） | +1（31→32） |
| `Mega_UndoLog_SparseWrite32_Commit` | ~（p=0.135） | +4.4% | +2（26→28） |

对 DeepCopy 基线（~228µs）仍有 ~28× 优势；若对 RemoveAt 热路径敏感，可回退方案 b（原地移位 + 文档禁止双根并存）。

## 附：验证命令

```bash
go build ./...
go vet ./...
go test ./...
go test ./cmd/undoproxy-gen ./internal/... ./examples/...   # examples 为独立 module，需进入目录实跑
cd examples/gamestore && go test ./...
cd examples/gamestore-migrate && go test ./...
```

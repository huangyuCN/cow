# COW 显式会话事务 API 设计

## 1. 背景

当前 `COW`（Copy-On-Write，写时复制）事务运行时已经具备：

- 根事务提交 / 回滚；
- 多层栈式 `Savepoint / RollbackTo`；
- 组件级与容器级写路径；
- benchmark 诊断结果。

最新一轮 benchmark 诊断已经明确：

- path-copy 写路径本身已经基本站住；
- 当前整体性能问题主要集中在**事务框架固定成本**；
- 其中 `Run` + `context.WithValue` + `FromContext` 这条 API 主线是主要嫌疑人之一。

因此，下一阶段不再只是局部性能优化，而是要**调整事务 API 的主模型**：

- 从“`Run(ctx, store, fn)` + `FromContext` + 包级 `Savepoint`”切换为
- “显式 `TxSession` + `Begin/Commit/Rollback` + 会话方法式 `Savepoint`”。

## 2. 本次设计目标

本次设计目标如下：

1. 将事务主模型切换为显式会话。
2. 彻底移除 `Run` / `FromContext` / 包级 `Savepoint` / 包级 `RollbackTo`。
3. 将 `Savepoint` 和写路径入口都挂到显式会话模型上。
4. 保持现有事务语义与 path-copy 写路径行为不退化。
5. 将测试、benchmark、文档全部迁移到新主模型。

## 3. 本次设计不覆盖的范围

本轮不覆盖以下内容：

- 再次重写 path-copy 内部结构；
- 新增对象池；
- 再做一轮 benchmark 诊断设计；
- 代码生成器；
- 宿主适配层；
- 仓储层集成策略。

也就是说：

- 这轮首先解决的是**事务 API 主线切换**；
- 不是顺带做一轮新的运行时结构重写。

## 4. 方案比较

### 方案 A：显式会话为核心，`Run` 变薄包装

- 核心 API 改成显式会话；
- `Run` 仍保留为便捷封装。

优点：

- 迁移温和；
- 旧代码与旧测试改动相对少。

缺点：

- `Run` 仍会长期留在主线认知里；
- benchmark 和文档会继续被双主模型污染。

### 方案 B：显式会话与旧 `context` 模型长期并列

- 新旧 API 一起保留。

优点：

- 兼容成本低。

缺点：

- 两套主模型长期共存；
- 后续维护成本和心智成本都更高。

### 方案 C：显式会话成为唯一主线，`Run/context` 彻底移除

- 主 API 直接切到显式 `TxSession`；
- `Run` / `FromContext` / 包级 `Savepoint` 全部从主线实现、测试、文档中清理。

优点：

- 性能主热路径最干净；
- API 责任边界最清楚；
- 后续对象池、延迟初始化、会话复用更自然。

缺点：

- 这轮迁移范围更大；
- 需要一次性完成测试、benchmark 和文档口径切换。

### 推荐方案

推荐采用**方案 C**。

原因：

- benchmark 已经说明框架层成本是主要问题；
- 若保留 `Run/context` 兼容层，后续主线很难真正变轻；
- 既然已经决定主模型切换，就应避免长期双轨并存。

## 5. 核心 API 设计

### 5.1 开始与结束事务

建议新的核心 API 为：

```go
func Begin[T any](store Store[T], clone func(*T) *T) (*TxSession[T], error)

func (s *TxSession[T]) Commit() error
func (s *TxSession[T]) Rollback()
```

语义如下：

- `Begin(...)` 创建一个活动事务会话，失败直接返回错误；
- `Commit()` 负责提交当前事务视图到 `Store`；
- `Rollback()` 负责放弃当前未提交状态，并结束事务；
- 一旦 `Commit()` 或 `Rollback()` 完成，会话进入 closed 状态。

### 5.2 Savepoint 变为会话方法

建议检查点 API 改为：

```go
func (s *TxSession[T]) Savepoint() (SavepointID, error)
func (s *TxSession[T]) RollbackTo(id SavepointID) error
```

语义如下：

- `Savepoint()` 在当前活动会话上创建检查点；
- `RollbackTo(id)` 在当前活动会话上回滚到目标检查点；
- 保持现有的栈式检查点语义；
- 会话关闭后，相关调用返回 `ErrSessionClosed`。

### 5.3 写路径入口

写路径主入口继续以显式会话为中心：

```go
bag := mutableBag(sess)
items := mutableBagItems(sess)
```

这意味着：

- 不再通过 `context` 找事务；
- 事务状态和写路径状态都围绕 `TxSession` 本身组织。

## 6. 错误处理风格

本轮采用传统事务风格：

- `Begin()` 失败直接返回错误；
- `Commit()` 返回错误；
- `Rollback()` 尽量无错；
- 框架误用和业务失败分开处理。

推荐错误约束如下：

- 会话已关闭后再次 `Commit()`：返回 `ErrSessionClosed`；
- 会话已关闭后继续写入：返回 `ErrSessionClosed`；
- 会话已关闭后继续 `Savepoint()` / `RollbackTo()`：返回 `ErrSessionClosed`；
- 非法检查点：返回 `ErrInvalidSavepoint`。

## 7. 内部状态结构

本轮不要求再重写 path-copy 内部表示。

因此，建议 `TxSession` 继续沿用当前已经跑通的核心状态：

- `base`
- `work`
- `dirty`
- `cloned`
- `checkpoints`
- `finished` 或 `closed`

本轮的重点是：

- 切换事务 API 主线；
- 而不是在同一轮里再次改动 path-copy 内部布局。

## 8. 旧入口的清理范围

这轮要求把旧入口清理得彻底，而不是降级保留。

需要清理的内容包括：

- 删除 `run.go`
- 删除 `context.go`
- 删除包级 `Savepoint(ctx)` / `RollbackTo(ctx, id)` 形式
- 删除或改写所有依赖 `Run(...)` 的测试
- 删除或改写所有依赖 `FromContext(...)` 的 benchmark
- 文档中不再把 `Run` 当作当前主 API

本轮不保留过渡兼容层。

## 9. 测试迁移策略

### 9.1 根事务语义测试

现有测试需要从：

- `Run commit`
- `Run rollback on error`
- `Run rollback on panic`

迁移为：

- `Begin -> 写 -> Commit`
- `Begin -> 写 -> Rollback`
- 如需覆盖 `panic`，由测试自己使用 `defer recover`，不再由运行时包裹。

### 9.2 Savepoint 测试

检查点测试全部切到方法式 API：

- `sp, err := sess.Savepoint()`
- `err := sess.RollbackTo(sp)`

### 9.3 写路径与共享语义测试

以下测试必须继续保留并通过：

- 修改 `Bag` 时，`Quest` 继续共享；
- `mutableBag` 负责组件级 materialize；
- `mutableBagItems` 负责容器级首次复制；
- `DirtySet` 与 `cloned` 语义不退化。

### 9.4 benchmark 迁移

benchmark 也必须完全迁移到显式会话模型：

- 框架层 benchmark：改为 `Begin/Commit/Rollback` 路径；
- 纯写路径 benchmark：直接构造 `TxSession`；
- 端到端 benchmark：不再出现 `Run/context` 路径。

## 10. 验收标准

本轮至少需要满足以下验收项：

1. 主 API 完成切换，仅保留显式会话模型；
2. `Run` / `FromContext` / 包级 `Savepoint` 清理彻底；
3. 提交 / 回滚 / 栈式 `Savepoint` 语义不退化；
4. 组件级与容器级写路径行为不退化；
5. benchmark 全部切换到显式会话模型。

## 11. 实施建议

建议按以下顺序实施：

1. 先写显式会话 API 的失败测试；
2. 删除旧入口，使调用点在编译层面强制迁移；
3. 实现 `Begin/Commit/Rollback` 与方法式 `Savepoint`；
4. 迁移现有事务测试和共享语义测试；
5. 迁移 benchmark 诊断口径；
6. 跑全量测试与 benchmark；
7. 再根据新 benchmark 结果决定下一轮是否继续优化框架层分配。

## 12. 结论

本轮设计的核心不是继续微调某个局部性能热点，而是把事务 API 的主热路径从 `Run/context` 模型切换到显式会话模型。

这轮完成后，后续性能优化才会建立在更干净的主线之上：

- 会话生命周期显式；
- 检查点方法显式；
- 写路径入口显式；
- benchmark 口径也不再被旧 `context` 模型污染。

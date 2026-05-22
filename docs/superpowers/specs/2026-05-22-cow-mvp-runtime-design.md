# COW MVP 运行时设计

## 1. 目标

本设计只覆盖 `COW`（Copy-On-Write，写时复制）事务框架的 `MVP` 运行时能力：

- `Run(ctx, store, fn)` 是唯一标准入口；
- 支持根事务提交/回滚；
- 支持多层栈式 `Savepoint`；
- 仅支持组件级受控写入；
- 提供简版 `DirtySet`；
- 不实现宿主适配层、代码生成器、手动事务绑定接口。

本设计用于把 [MVP_REQUIREMENTS.md](/Users/huangyu/work/golang/src/cow/MVP_REQUIREMENTS.md) 中已经确认的需求，进一步收敛为可实现的运行时契约。

## 2. Store 契约

`Store` 只承担已提交根状态的读取与替换，不承担锁、事务嵌套、宿主调度或持久化职责。

```go
type Store[T any] interface {
    Load() *T
    Commit(next *T)
}
```

约束如下：

- `Load()` 返回当前已提交根状态；
- `Commit(next)` 以一次根替换的方式提交新状态；
- `Store` 的正确使用依赖宿主保证“同一聚合串行进入事务”；
- `MVP` 可先提供一个内存实现，供测试和 benchmark 使用。

## 3. Session 形态

`TxSession` 表示一次根事务的运行时状态。

```go
type TxSession[T any] struct {
    store       Store[T]
    base        *T
    work        *T
    clone       func(*T) *T
    checkpoints []checkpoint[T]
    dirty       DirtySet
    committed   bool
    finished    bool
}
```

字段语义如下：

- `store`：当前事务绑定的已提交根存储；
- `base`：事务开始时看到的已提交根；
- `work`：当前事务的工作根视图；
- `clone`：根快照克隆函数，供初始化和 `Savepoint` 回滚恢复使用；
- `checkpoints`：检查点栈；
- `dirty`：当前事务内已修改组件集合；
- `committed`：标记根事务是否已经提交；
- `finished`：标记事务是否已经结束，防止结束后继续操作。

检查点只用于栈式局部回滚：

```go
type SavepointID uint64

type checkpoint[T any] struct {
    id    SavepointID
    root  *T
    dirty DirtySet
}
```

## 4. 受控写入模型

- 业务代码不直接修改 `store.Load()` 返回值。
- 运行时通过 `Run` 创建 `TxSession`，并通过 `context.Context` 暴露给业务读取。
- 示例组件通过手写访问器确保首次写入时拷贝组件及其可变字段。
- `Savepoint` 后的回滚通过恢复根快照与 `DirtySet` 快照实现。
- `RollbackTo` 只允许回滚到当前栈顶检查点，不触发 `Store.Commit`，也不结束根事务。
- 事务结束后不得继续使用事务内获得的可变引用。

推荐的最小访问形态如下：

```go
func Run[T any](
    ctx context.Context,
    store Store[T],
    clone func(*T) *T,
    fn func(context.Context) error,
) (err error)

func FromContext[T any](ctx context.Context) (*TxSession[T], bool)

func Mutable[T any, C any](sess *TxSession[T], pick func(root *T) *C) *C
```

## 5. 错误模型

`MVP` 先定义以下公共错误：

```go
var (
    ErrNoSession        = errors.New("cow: no active session")
    ErrSessionClosed    = errors.New("cow: session already finished")
    ErrInvalidSavepoint = errors.New("cow: invalid savepoint")
)
```

语义如下：

- 缺失事务上下文：返回 `ErrNoSession`；
- 已结束事务继续操作：返回 `ErrSessionClosed`；
- 非法检查点：返回 `ErrInvalidSavepoint`。

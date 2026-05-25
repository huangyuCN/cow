# COW 代码生成 TxPlayer Overlay 事务视图设计

## 1. 背景

当前仓库中的 `COW`（Copy-On-Write，写时复制）运行时已经完成：

- 显式 `TxSession` 主模型切换；
- `Begin()` 默认只读、首次写升级；
- 大根稀疏写 benchmark；
- 只读事务 benchmark。

最近几轮 benchmark 已经说明：

- 当前 path-copy / lazy session 路线在“大根稀疏写”和“完全只读事务”场景下都成立；
- 但这条路线在实现形态上，仍然与目标中的“字段级读穿透 + 写时字段接管”存在明显差异。

用户希望的目标模型不是：

- 围绕一个通用 `TxSession[T]` 去维护根级工作副本，

而是更接近：

- 为具体业务类型生成一个专用事务视图，例如 `TxPlayer`；
- `TxPlayer` 持有原始 `*Player`；
- 所有业务都通过 `TxPlayer` 的生成方法访问字段；
- 读时优先读事务槽位，未命中则直接读 `base`；
- 写时首次把该字段从 `base` lazy copy 到 `tx`；
- 提交时只按字段级 dirty 状态重建一个新的顶层 `Player`。

因此，这一轮不是继续给当前 path-copy 路线做局部补丁，而是要明确设计一个新的事务视图原型方向，用来验证：

- 代码生成的字段级 overlay（覆盖层）事务模型，是否更贴近目标语义与潜在性能上限。

## 2. 本次设计目标

本轮只覆盖以下目标：

1. 为一个具体类型 `Player` 设计类型专用事务视图 `TxPlayer`。
2. 全部字段访问都通过生成方法完成，不允许直接操作字段。
3. 第一版只处理 `Player` 的直接字段。
4. 第一版字段类型只支持：
   - 标量
   - `map`
   - `slice`
5. 读时回退到 `base`，写时首次 copy 该字段。
6. `Commit()` 只重建一个新的顶层 `Player`，不做整根递归深拷贝。
7. 不提交时直接丢弃 `TxPlayer`，第一版不提供 `Rollback()`。

## 3. 本次设计不覆盖的范围

本轮不覆盖以下内容：

- 通用代码生成框架；
- 任意结构体类型生成；
- 嵌套对象递归事务视图；
- 运行时反射模型；
- `interface` / 任意复杂字段类型；
- 并发冲突检测；
- `Savepoint`；
- 原地回写原对象；
- `slice` 的元素级 overlay；
- 通用事务调度器。

也就是说：

- 这轮首先验证的是一个**类型专用、单层字段、代码生成**的 overlay 原型；
- 不是立刻替代现有通用事务框架。

## 4. 方案比较

### 方案 A：类型专用、单层字段、代码生成的 `TxPlayer`

- 只为 `Player` 生成：
  - `TxPlayer`
  - `BeginPlayer(base *Player) *TxPlayer`
  - `(*TxPlayer).Commit() *Player`
- 只处理直接字段；
- 字段访问全部通过生成方法；
- `map` / `slice` 都按字段级整体首次 copy。

优点：

- 最贴近目标模型；
- 读写语义最清楚；
- 最适合作为第一版原型验证。

缺点：

- 只能支持单类型；
- 还不是通用框架。

### 方案 B：类型专用，但继续套在当前 `TxSession` 外壳里

- 生成 `TxPlayer`，但外面仍套当前通用事务壳。

优点：

- 可复用现有框架的一部分能力。

缺点：

- 会混合两套模型；
- 不利于隔离验证“字段级 overlay”本身的价值；
- 结果更难解释。

### 方案 C：直接做通用代码生成框架

- 一开始就支持任意结构体；
- 生成 `TxXxx`；
- 再配合统一 runtime / registry。

优点：

- 长期上限最高。

缺点：

- 第一版复杂度过大；
- 很容易在性能验证前就把生成器做成半个编译器。

### 推荐方案

推荐采用**方案 A**。

原因：

- 当前最需要验证的不是“通用生成框架能不能做”，而是“你想要的这套字段级 overlay 模型值不值得做”；
- 方案 A 最能把这个问题独立、干净地测清楚。

## 5. 第一版 API 设计

第一版采用类型专用 API：

```go
func BeginPlayer(base *Player) *TxPlayer

func (tx *TxPlayer) Commit() *Player
```

第一版**不提供**：

```go
func (tx *TxPlayer) Rollback()
```

原因是：

- `TxPlayer` 本身就是一个临时覆盖层；
- `base *Player` 不会被原地修改；
- 如果不提交，直接丢弃 `TxPlayer` 即可；
- `Rollback()` 不增加额外语义价值，反而让第一版 API 变重。

因此第一版事务语义是：

- `BeginPlayer(base)`：创建覆盖层事务视图；
- `Commit()`：返回一个新的 `*Player`；
- 不提交：直接丢弃 `TxPlayer`。

## 6. TxPlayer 内部状态结构

### 6.1 基础结构

`TxPlayer` 至少包含：

1. `base *Player`
2. 每个字段一份事务槽位
3. 每个字段一个 `has/dirty` 标记

一个简化示意如下：

```go
type TxPlayer struct {
    base *Player

    name    string
    hasName bool

    level    int
    hasLevel bool

    items    map[int]int
    hasItems bool

    skills    []int
    hasSkills bool
}
```

语义如下：

- `base`：原始 `Player`；
- `hasX == false`：表示该字段尚未被事务接管；
- `hasX == true`：表示该字段已进入事务视图，读写都走 tx 槽位。

### 6.2 标量字段

对于标量字段：

- 读时：
  - `hasX == false`，返回 `base.X`
  - `hasX == true`，返回 `tx.x`
- 写时：
  - 直接写 `tx.x`
  - 将 `hasX = true`

这里必须有独立标记，不能依赖零值判断，因为：

- 零值并不表示“未写过”。

### 6.3 map 字段

对于 `map` 字段：

- 读时：
  - 若 `hasItems == false`，直接从 `base.Items` 读取；
  - 若 `hasItems == true`，从 `tx.items` 读取。
- 首次写时：
  - `maps.Clone(base.Items)` 到 `tx.items`
  - 置 `hasItems = true`
- 后续写时：
  - 只操作 `tx.items`

### 6.4 slice 字段

对于 `slice` 字段：

- 读时：
  - 若 `hasSkills == false`，读 `base.Skills`
  - 若 `hasSkills == true`，读 `tx.skills`
- 首次写时：
  - 整体 copy 一份 `slice` 到 `tx.skills`
  - 置 `hasSkills = true`
- 后续写时：
  - 只操作 `tx.skills`

第一版**不做**：

- `slice` 元素级 overlay；
- 插入 / 删除 / 重排的细粒度增量表示。

第一版对 `slice` 的语义收敛为：

- 首次写该字段时整体 copy，后续直接在 tx slice 上操作。

## 7. 生成方法语义

### 7.1 总体原则

第一版所有字段访问都通过生成方法完成，不暴露字段本身。

也就是说：

- 不允许业务直接读写 `TxPlayer` 内部字段；
- 容器字段尤其不能直接裸露底层 `map` / `slice` 给业务方绕开语义。

### 7.2 标量字段方法

标量字段建议生成：

- 读方法
- 写方法

例如：

```go
func (tx *TxPlayer) Name() string
func (tx *TxPlayer) SetName(v string)
```

### 7.3 map 字段方法

`map` 字段不建议直接生成：

```go
func (tx *TxPlayer) Items() map[int]int
```

更合适的是受控操作方法，例如：

```go
func (tx *TxPlayer) Item(id int) (int, bool)
func (tx *TxPlayer) SetItem(id int, v int)
func (tx *TxPlayer) DeleteItem(id int)
```

这样才能确保：

- 未命中时回退到 `base`
- 首次写时先 clone
- 后续写只落到 tx map

### 7.4 slice 字段方法

`slice` 字段同样不建议直接裸露底层切片。

更合适的是生成受控方法，例如：

```go
func (tx *TxPlayer) Skill(i int) (int, bool)
func (tx *TxPlayer) SkillCount() int
func (tx *TxPlayer) AppendSkill(v int)
func (tx *TxPlayer) SetSkill(i int, v int)
```

第一版重点不是接口尽可能全面，而是：

- 保证事务语义不会被绕开；
- 让生成方法完全掌控何时 copy、何时读 `base`、何时写 `tx`。

## 8. Commit 语义

### 8.1 顶层对象重建

第一版 `Commit()` 采用：

- 只重建一个新的顶层 `Player`

而不是：

- 对整个对象图做递归 deep copy。

伪代码示意：

```go
func (tx *TxPlayer) Commit() *Player {
    base := tx.base
    out := &Player{}

    if tx.hasName {
        out.Name = tx.name
    } else {
        out.Name = base.Name
    }

    if tx.hasItems {
        out.Items = tx.items
    } else {
        out.Items = base.Items
    }

    if tx.hasSkills {
        out.Skills = tx.skills
    } else {
        out.Skills = base.Skills
    }

    return out
}
```

### 8.2 核心语义

提交时的关键规则是：

- 未修改字段：直接复用 `base` 中的值；
- 已修改标量字段：使用 tx 槽位值；
- 已修改 `map` 字段：使用已 lazy copy 的 tx map；
- 已修改 `slice` 字段：使用已整体 copy 的 tx slice。

因此这不是：

- 提交时做一次整根 `DeepCopy`

而是：

- 提交时做一次**字段级 merge / 顶层对象组装**。

### 8.3 第一版不做的提交能力

第一版不做：

- 并发版本检查；
- 冲突检测；
- 原地回写 `base`；
- 自动在“回写 base / 重建顶层对象”之间动态选择最优路径。

第一版首先要保证的是：

- 语义正确；
- 结构简单；
- 容易 benchmark。

## 9. 第一版范围边界

### 9.1 支持内容

第一版只支持：

- 一个具体类型：`Player`
- 只处理 `Player` 的直接字段
- 字段类型仅限：
  - 标量
  - `map`
  - `slice`

### 9.2 不支持内容

第一版明确不支持：

- 递归子对象事务视图
- 嵌套 struct 指针事务化
- `interface` 字段语义扩展
- 通用 `TxXxx` 生成器框架
- 多类型统一 runtime

这意味着：

- 第一版是一个受控原型；
- 不是完整终态框架。

## 10. 验收方式

### 10.1 语义验收

至少需要验证：

- 未修改字段读取时直接回退到 `base`
- 首次写字段时才发生 copy
- `map` / `slice` 写后不污染 `base`
- `Commit()` 后：
  - 修改过的字段来自 tx
  - 未修改字段继续复用 `base`

### 10.2 代码生成验收

至少需要验证：

- 能为 `Player` 生成 `TxPlayer`
- 生成代码可编译
- 业务侧不需要直接操作字段

### 10.3 benchmark 验收

第一版至少补三类 benchmark：

1. 完全只读
2. 稀疏写少量字段
3. 同事务重复写同一字段 / 同一容器

目标不是先证明它一定替代当前路径，而是先回答：

- 这套“字段级覆盖层 + 顶层 merge”模型，相比当前 path-copy 路线是否有潜在优势。

## 11. 定位结论

这一轮不应把该设计定义成：

- 现有事务框架的直接替代方案

而应定义成：

- 一个新的事务模型原型分支 / 代码生成 overlay 实验。

原因是：

- 当前还没有证据证明它一定比现有路线更好；
- 但从目标语义和潜在性能路径看，它明显更贴近真正想要的系统形态。

## 12. 结论

下一阶段不应继续只在当前 path-copy `TxSession` 上做局部修补，而应独立验证一个新的字段级 overlay 原型：

- 类型专用；
- 单层字段；
- 标量 / `map` / `slice`；
- 字段访问完全方法化；
- 读穿透 `base`；
- 写时字段级首次 copy；
- 提交时只重建顶层 `Player`。

只有先把这个原型跑通并 benchmark，才能更有把握地回答：

- 这条更贴近目标语义的设计，是否值得继续演进成下一代事务模型。

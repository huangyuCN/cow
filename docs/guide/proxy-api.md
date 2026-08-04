# 生成写代理 API

## 概述

生成方法挂在**原业务类型**上，签名均含 `ctx *TxContext`。命名规则由字段类型决定（map/slice/指针/标量）。以下以本仓库 `Player` / `Hero` 为例。

## 标量 Put

```go
player.PutAssets(ctx, "gold", 500)
h.PutLevel(ctx, 2)
```

回滚：恢复写入前的值。

## map Put

```go
h.PutSkills(ctx, skillID, skill)
player.PutHeros(ctx, heroID, heroPtr)
```

- key 已存在：恢复旧值  
- key 不存在：回滚时 `delete`

**注意**：`PutHeros(ctx, k, nil)` 表示 map 槽位存 **nil 指针**，key 仍存在；删 key 请用 `RemoveHeros`（见下），不要用 Put 传 nil 代替 delete。

## map Remove（删 key）

```go
player.RemoveHeros(ctx, heroID)
player.RemoveAssets(ctx, "gold")
```

语义等价 `delete(map, key)`：nil map 或 key 不存在时为 **no-op**（不记录 undo）。与 `Put*` 严格分离。

## 指针 Set（整槽替换）

```go
player.SetMainHero(ctx, newHero)
player.SetMainHero(ctx, nil) // 清空指针
```

替换整个 `*Struct` 指针（**直接赋值**，不 `CloneForWrite`，`p.MainHero` 与传入的 `val` 为同一指针）；与 `GetMainHeroForWrite` 并存——Get 用于就地 COW 改子字段，Set 用于换整棵子树或清空。若 `val` 仍被外部共享且会裸写子字段，应自行 `CloneForWrite` 后再 Set，或改用 `Get*ForWrite`。

## slice Append / Set / Remove / Truncate

```go
player.AppendItems(ctx, item)
```

回滚：截断到原 `len` 或恢复被替换元素（视生成方法而定）。

## 指针与 map 元素：Get*ForWrite

延迟局部拷贝（Clone）：仅在被写子结构时克隆并替换指针/map 槽位。

```go
h := player.GetMainHeroForWrite(ctx)
if h != nil {
	h.PutLevel(ctx, 2)
}
```

`map[K]*Struct` 按**字段名**生成 `Get{Field}ForWrite`，例如 `player.GetHerosForWrite(ctx, heroID)`，再 `h.GetSkillsForWrite(ctx, skillID)`。  
**破坏性变更：** 旧 `GetHeroForWrite`（字段 `Heros`）已更名为 `GetHerosForWrite`；同类字段不再用元素类型单数名。

## CloneForWrite

每个纳入类型图的 struct 生成 `CloneForWrite()`，供 `Get*ForWrite` 内部使用；业务一般通过 `Get*ForWrite` 间接使用。

**fork 语义（浅拷贝）**：`CloneForWrite` 只复制 struct 本体，map/slice/指针字段与原根**共享底层结构**。若业务手动 fork 并保留原根：

- `Remove*At` 为写时复制，不会移位污染原根；Rollback 后原根保持原状。
- `Set*At` / `Append*` 仍写入共享底层结构：写入在 Rollback 前对原根可见，Rollback 会恢复值，但事务进行中两个根会互相可见。
- 推荐始终维持**单活根**模型（同一时刻只有一个根接受写），fork 用例见 `undo_semantics_test.go`。

## 完整稀疏写示例

与 benchmark / 测试一致的三处写：

```go
func applySparseWrites(p *Player, ctx *TxContext) {
	p.PutAssets(ctx, "gold", 500)
	p.AppendItems(ctx, newTestItem(9999, "Shield"))
	h := p.GetMainHeroForWrite(ctx)
	if h != nil {
		h.PutLevel(ctx, 2)
	}
}
```

源码：`bench_fixture_test.go`。

可运行：`doc_examples_test.go` 中 `ExamplePlayer_sparseWrite`。

## 类型别名

同包内可为 map/slice 定义别名，例如 `type Equips map[int64]*Equip`，生成器会生成 `PutEquips` / `RemoveEquips` / `GetEquipForWrite`，`make(Equips)` 保留别名类型名。

## 边界

- 必须通过代理写；直接 `p.Level = 1` 或 `delete(p.Heros, k)` 会被 `cowbarewrite` 拒绝（初始化/白名单除外）。
- `json.Unmarshal` 等反射写静态不可见；Unmarshal 后仍须走代理（见 [limitations.md](limitations.md)）。
- **生成 API 不做边界检查**：越界下标（`Set*At`/`Remove*At`）或缺失 key（`Get*ForWrite` 返回 nil 后继续解引用）会 panic，调用方须自行保证；这是刻意取舍，换取零防御开销的热路径。

## 相关链接

- [tx-context.md](tx-context.md)
- [bare-write-guard.md](bare-write-guard.md)
- 维护：[../../cmd/undoproxy-gen/README.md](../../cmd/undoproxy-gen/README.md)

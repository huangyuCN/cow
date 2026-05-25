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
```

- key 已存在：恢复旧值  
- key 不存在：回滚时 `delete`

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

`map[K]*Struct` 形如 `GetSkillForWrite(ctx, k)`（见 `zz_generated.undo_proxy.go`）。

## CloneForWrite

每个纳入类型图的 struct 生成 `CloneForWrite()`，供 `Get*ForWrite` 内部使用；业务一般通过 `Get*ForWrite` 间接使用。

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

## 边界

- 必须通过代理写；直接 `p.Level = 1` 会被 `cowbarewrite` 拒绝（初始化/白名单除外）。
- `json.Unmarshal` 等反射写静态不可见；Unmarshal 后仍须走代理（见 [limitations.md](limitations.md)）。

## 相关链接

- [tx-context.md](tx-context.md)
- [bare-write-guard.md](bare-write-guard.md)
- 维护：[../../cmd/undoproxy-gen/README.md](../../cmd/undoproxy-gen/README.md)

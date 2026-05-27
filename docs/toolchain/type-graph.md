# 类型图规则

`undoproxy-gen` 与 `undocheck` 对「受监控类型集合」使用同一套规则，避免「生成了代理却未检查裸写」或相反。

## 根类型

struct 上标记：

```go
// +cow:undoproxy-gen=true
type Player struct { ... }
```

## 同包可达嵌套

从根类型字段出发，**同一 Go 包内**引用的 struct 类型纳入图中（具体遍历由 `internal/cowgen.BuildGraph` 实现）。

## 实现入口

| 组件 | 包 | 职责 |
|------|-----|------|
| 生成 | `internal/cowgen` | `BuildGraph` → 模板 emit |
| 分析 | `internal/cowmon` | `BuildFromSyntax` / `LoadMonitored` → `MonitoredSet` |
| 裸写检测 | `cmd/undocheck` | AST + `MonitoredSet` |
| 改写目录 | `internal/cowproxy` | 与生成器共享字段分类的 `RewriteCatalog` |

## 类型别名

字段声明为同包 **map/slice 别名**（如 `type Equips map[int64]*Equip`）时，分类与字面 `map[...]` 相同，生成签名与 `make()` 使用别名名（`Equips`、`ItemList` 等）。底层为 struct 的别名嵌套不在支持范围。

## 不支持

- 跨包嵌套 struct 作为字段类型
- `interface{}`、channel、func 作为 map 值或 slice 元素

## 相关链接

- [README.md](README.md)
- [../../cmd/undoproxy-gen/README.md](../../cmd/undoproxy-gen/README.md)
- [../../cmd/undocheck/README.md](../../cmd/undocheck/README.md)

# 限制与非目标

## 并发与宿主

- **无** `TxContext` / 聚合根的跨 goroutine 并发写支持。
- 宿主须保证对单聚合根串行写；cow 不替代 Actor 调度或分布式锁。

## 代码生成范围

- `undoproxy-gen`：**同包**类型图；根类型 `+cow:undoproxy-gen=true` + 同包可达嵌套 struct。
- 不支持：`interface{}`、channel、func 作为受监控 map/slice 元素类型。
- 跨包嵌套字段：不生成、不监控。

## 静态分析

- `undocheck` 仅编译期；**看不到** `json.Unmarshal`、`reflect` 等运行期写入。
- 约定：反序列化填充后，业务逻辑仍只通过代理修改；或采用 DTO 分层。

## 运行模型

- 不提供「每请求自动 DeepCopy 副本」；DeepCopy 生成物仅用于性能对照（见 [overview.md](overview.md)）。
- 不捆绑 HTTP/gRPC/消息中间件；`TxContext` 注入方式由宿主实现。
- **单活根契约**：`CloneForWrite` 是浅拷贝 fork，map/slice/指针与原根共享底层结构；`Remove*At` 写时复制不污染原根，但 `Set*At`/`Append*` 在事务进行中对双根互相可见。推荐同一时刻只有一个根接受写（详见 [proxy-api.md](proxy-api.md) 的 CloneForWrite 小节）。
- **生成 API 无边界检查**：越界/缺失 key 会 panic，由调用方保证（见 [proxy-api.md](proxy-api.md) 边界小节）。
- **Undo 内存与类型图规模线性相关**：`undoOp` 为每个可达 struct 携带指针槽、为每类 key/标量/slice/内层 map 携带一个槽，`push`/`Reset` 为整 struct 值拷贝/清零。大型聚合根（百级字段图）下单条 Undo 约数百字节，超大类型图请结合 benchmark 评估。

## 工具

- `undorewrite` 不保证 100% 自动改写；复杂表达式需人工修复。
- 运行期裸写检测：非目标（未来若有增强，亦不在当前范围内）。

## 相关链接

- [overview.md](overview.md)
- [integration-checklist.md](integration-checklist.md)
- 设计档案：[../superpowers/specs/](../superpowers/specs/)

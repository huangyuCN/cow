# cow

**单协程聚合根 Undo Log 写代理**：业务失败时倒序执行逆操作回滚；成功路径不拷贝聚合根数据，仅清空 `TxContext` 日志。

模块路径：`github.com/huangyuCN/cow`

## 解决的问题

- **补偿脆弱**：请求处理链变长后，手写回滚/补偿易漏，难测试。
- **DeepCopy 成本高**：每次请求对整棵聚合根做全量深拷贝，在大 `map`/`slice` 场景下 CPU、分配与 GC 压力大。

## 前提

宿主对同一聚合根提供 **单 goroutine 串行写**（或等价保证）。`TxContext` **不加锁**，不可跨 goroutine 共享。

## 能力边界

- 不提供 `TxContext` 并发安全。
- 不提供运行期裸写检测（仅静态分析器 `undocheck` / `cowbarewrite`）。
- `undoproxy-gen` 初版仅支持 **同包** 类型图；容器元素不支持 `interface{}`、channel、func。
- 不捆绑具体 Actor / HTTP / 消息框架；文档提供接入模式，由宿主嵌入。

## 快速开始

```bash
# 引入模块（或 go.work replace 本地路径）
go get github.com/huangyuCN/cow@latest

# 1. 在聚合根上标记并生成代理
go install ./cmd/undoproxy-gen
# types.go: // +cow:undoproxy-gen=true
go generate ./...

# 2. 静态禁止裸写
go install ./cmd/undocheck
go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
```

单次请求作用域示例（失败回滚）：

```go
ctx := txPool.Get().(*TxContext)
ctx.Reset()
defer func() {
    if err != nil {
        ctx.Rollback()
    }
    txPool.Put(ctx)
}()
player.PutAssets(ctx, "gold", 500)
// ...
```

成功提交：业务无错时 `ctx.Reset()` 清空日志即可（见 [docs/guide/tx-context.md](docs/guide/tx-context.md)）。

## 文档

| 文档 | 说明 |
|------|------|
| [docs/README.md](docs/README.md) | 文档总索引（含贡献者阅读顺序） |
| [docs/guide/](docs/guide/) | **集成方**功能手册（推荐从此入手） |
| [docs/toolchain/](docs/toolchain/) | **维护者**工具链说明 |
| [cmd/](cmd/) | 各子命令 README |
| [CONTRIBUTING.md](CONTRIBUTING.md) | 贡献流程与约定 |
| [docs/superpowers/](docs/superpowers/) | **维护者**设计 spec / plan / benchmark 档案 |

## License

Apache License 2.0，见 [LICENSE](LICENSE)。

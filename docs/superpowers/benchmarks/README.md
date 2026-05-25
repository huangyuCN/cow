# Benchmark 归档约定

本目录存放**经确认**的 benchmark 对比日志（非临时 `.txt` 原始输出）。

## 每条记录须包含

- 日期、主题、关联 spec/plan 链接
- `go version`、OS/CPU、`GOMAXPROCS`、git **commit**（工作区未提交时注明）
- 完整 `go test -bench=...` 命令
- Markdown 对比表：`ns/op`、`B/op`、`allocs/op`，及相对基线变化（若有上一次归档）

## 文件命名

`cow-<主题>-benchmark.md`，按主题追加章节，勿覆盖历史 run。

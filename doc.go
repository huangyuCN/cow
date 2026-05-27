// Package cow 提供单协程聚合根 Undo Log 写代理。
//
// 裸写静态检查（需先安装分析器）：
//
//	go install ./cmd/undocheck
//	go vet -vettool=$(go env GOPATH)/bin/undocheck ./...
//
// 历史裸写批量改写（默认 dry-run，确认后加 -w）：
//
//	go install ./cmd/undorewrite
//	undorewrite ./yourpkg/...
//
// 完整说明见仓库 README.md 与 docs/guide/。
//
// +k8s:deepcopy-gen=package
// +cow:undoproxy-gen=package
// +groupName=cow.huanghaiyu.cn
package cow

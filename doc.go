// Package cow 提供单协程聚合根 Undo Log 写代理（MVP 验证）。
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
// +k8s:deepcopy-gen=package
// +cow:undoproxy-gen=package
// +groupName=cow.huanghaiyu.cn
package cow

package main

import (
	"fmt"

	"github.com/huangyuCN/cow/examples/gamestore"
)

func main() {
	fmt.Println("=== cow examples/gamestore ===")
	fmt.Println("rollback demo:")
	gamestore.RunDemoRollback()
	fmt.Println("commit demo:")
	gamestore.RunDemoCommit()
	fmt.Println("done — run: go test ./...")
}

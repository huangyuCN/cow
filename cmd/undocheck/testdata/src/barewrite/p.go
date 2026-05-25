package barewrite

// Player 测试用聚合根。
//
// +cow:undoproxy-gen=true
type Player struct {
	Level int32
	Items []*Item
}

// Item 嵌套类型。
type Item struct {
	Name string
}

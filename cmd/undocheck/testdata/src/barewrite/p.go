package barewrite

// Player 测试用聚合根。
//
// +cow:undoproxy-gen=true
type Player struct {
	Level int32
	Items []*Item
	Heros map[int32]*Hero
}

type Hero struct {
	Level int32
}

// Item 嵌套类型。
type Item struct {
	Name string
}

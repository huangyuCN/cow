package beforeshop

// Player 是裸写时代的聚合根示例类型（无 cow 生成 tag）。
type Player struct {
	Gold     int64
	Wallet   map[string]int64
	Items    []*Item
	MainHero *Hero
}

type Item struct {
	Id   int64
	Name string
}

type Hero struct {
	Level int32
}


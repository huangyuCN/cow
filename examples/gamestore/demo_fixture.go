package gamestore

// 本文件为夹具与构造辅助（undocheck 白名单：*_fixture.go）。

// NewDemoPlayer 构造演示用 Player（中等规模，保证 map/slice 非空）。
func NewDemoPlayer() *Player {
	p := &Player{
		Gold: 1000,
		Wallet: map[string]int64{
			"gold": 500,
			"gems": 10,
		},
		MainHero: &Hero{Level: 1, Skills: map[int32]*Skill{1: {Level: 1}}},
		Heros:    make(map[int32]*Hero),
		Bags:     make(map[int32][]*Item),
		Stats:    make(map[int32]map[string]int64),
	}
	for i := int32(1); i <= 5; i++ {
		p.Heros[i] = &Hero{Level: i, Skills: map[int32]*Skill{i: {Level: 1}}}
	}
	p.Items = []*Item{{Id: 1, Name: "starter"}}
	p.Bags[1] = []*Item{{Id: 101, Name: "bag1"}}
	p.Stats[1] = map[string]int64{"atk": 10, "def": 5}
	return p
}

// NewDemoGuild 构造演示用 Guild。
func NewDemoGuild() *Guild {
	return &Guild{
		Members: map[int32]*Member{
			1: {Name: "alice", Rank: 1},
		},
	}
}

func newLootItem() *Item       { return &Item{Id: 9001, Name: "loot"} }
func newUpgradedItem() *Item   { return &Item{Id: 1, Name: "upgraded"} }
func newBagDropItem() *Item    { return &Item{Id: 202, Name: "bag_drop"} }
func newMemberBob() *Member    { return &Member{Name: "bob", Rank: 2} }
func newTempMember() *Member   { return &Member{Name: "temp", Rank: 9} }

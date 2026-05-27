package gamestore

// +cow:undoproxy-gen=true
type Player struct {
	Gold     int64
	Wallet   map[string]int64
	Items    []*Item
	MainHero *Hero
	Heros    map[int32]*Hero
	Bags     map[int32][]*Item
	Stats    map[int32]map[string]int64
}

// +cow:undoproxy-gen=true
type Guild struct {
	Members map[int32]*Member
}

// Item 背包条目。
type Item struct {
	Id   int64
	Name string
}

// Hero 英雄子结构。
type Hero struct {
	Level  int32
	Skills map[int32]*Skill
}

// Member 公会成员。
type Member struct {
	Name string
	Rank int32
}

// Skill 技能。
type Skill struct {
	Level int32
}

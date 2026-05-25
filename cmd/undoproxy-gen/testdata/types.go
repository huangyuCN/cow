// Package testdata 供 undoproxy-gen 黄金测试。
package testdata

// +cow:undoproxy-gen=true
type Player struct {
	Gold  int64
	Items []*Item
	Loot  map[int32][]int32
	Buffs map[int32]map[string]int64
}

type Item struct {
	Id int64
}

// +cow:undoproxy-gen=true
type Room struct {
	Heros map[int32]*Hero
}

type Hero struct {
	Skills map[int32]*Skill
}

type Skill struct {
	Level int32
}

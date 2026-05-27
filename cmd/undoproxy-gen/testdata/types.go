// Package testdata 供 undoproxy-gen 黄金测试。
package testdata

// +cow:undoproxy-gen=true
type Player struct {
	Gold     int64
	Items    []*Item
	MainHero *Hero
	Heros    map[int32]*Hero
	Loot     map[int32][]int32
	Buffs    map[int32]map[string]int64
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
	// Power 用于验证 float32 标量 undo 槽位生成。
	Power float32
}

type Equips map[int64]*Equip

type ItemList []*Item

type Equip struct {
	Slot int32
}

// +cow:undoproxy-gen=true
type EquipBack struct {
	Equips Equips
	Spares ItemList
}

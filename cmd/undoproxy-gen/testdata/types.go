// Package testdata 供 undoproxy-gen 黄金测试。
package testdata

import "github.com/huangyuCN/cow/cmd/undoproxy-gen/testdata/leafpkg"

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

type Condition struct {
	Val int64
}

// +cow:undoproxy-gen=true
type Alpha struct {
	Conditions map[int32]map[string]*Condition
}

// +cow:undoproxy-gen=true
type Beta struct {
	Conditions map[int32]map[string]*Condition
}

type StackableItem struct {
	Qty int64
}

// +cow:undoproxy-gen=true
type Bag struct {
	Stackable        map[uint64]*StackableItem
	VirtualStackable map[uint64]*StackableItem
}

// +cow:undoproxy-gen=true
type NestBag struct {
	Primary   map[int32]map[string]*Condition
	Secondary map[int32]map[string]*Condition
}

type Cat int32

type Node struct {
	V int64
}

// +cow:undoproxy-gen=true
type TypedSlotsRoot struct {
	ByInt  map[int]map[string]*Node
	Score  Cat
	InnerA map[int32]map[string]int64
	InnerB map[int32]map[string]*Node
}

// +cow:undoproxy-gen=true
type ImportLeafRoot struct {
	ByEnum map[leafpkg.Enum]int64
	Tag    leafpkg.Enum
}

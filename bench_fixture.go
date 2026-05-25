package cow

import "fmt"

// newTestItem 夹具内构造 Item（仅 *_fixture.go 允许复合字面量写字段）。
func newTestItem(id int64, name string) *Item {
	return &Item{Id: id, Name: name}
}

// newTestItemsByID 按 Id 列表构造 Items。
func newTestItemsByID(ids ...int64) []*Item {
	out := make([]*Item, len(ids))
	for i, id := range ids {
		out[i] = &Item{Id: id}
	}
	return out
}

// newTestHeroProbe99 探针用 Hero。
func newTestHeroProbe99() *Hero {
	return &Hero{
		HeroId: 99,
		Level:  5,
		Skills: map[int32]*Skill{1: {SkillId: 1, Level: 1}},
	}
}

// newTestMailPut 探针 PutMails 用。
func newTestMailPut() *Mail {
	return &Mail{Id: 2, Subject: "put_mail", Body: "body"}
}

// newTestQuestPut 探针 PutQuests 用。
func newTestQuestPut() *Quest {
	return &Quest{Id: 2, State: 8, Objectives: map[int32]int32{0: 1}}
}

// newTestItemsForBagPut 探针 PutBags 用。
func newTestItemsForBagPut() []*Item {
	return []*Item{newTestItem(70005, "bag_put")}
}

// newPlayerForDeepCopyTest DeepCopy 隔离测试种子。
func newPlayerForDeepCopyTest() *Player {
	return &Player{
		Uid:      1,
		Assets:   map[string]int64{"gold": 1},
		MainHero: &Hero{Level: 1},
	}
}

// newPlayerWithItems 测试用小号 Player（字段构造仅在 fixture 白名单内）。
func newPlayerWithItems(items []*Item) *Player {
	p := &Player{MainHero: &Hero{HeroId: 1, Level: 1}}
	p.Items = items
	return p
}

func newBenchPlayer() *Player {
	assets := make(map[string]int64, 100)
	assets["gold"] = 1000
	assets["diamond"] = 100
	for i := 0; i < 98; i++ {
		assets[fmt.Sprintf("token_%d", i)] = int64(i + 1)
	}
	items := make([]*Item, 0, 500)
	for i := 0; i < 500; i++ {
		items = append(items, &Item{Id: int64(i + 1), Name: fmt.Sprintf("item_%d", i)})
	}
	return &Player{
		Uid:    10001,
		Level:  1,
		Assets: assets,
		Items:  items,
		MainHero: &Hero{HeroId: 99, Level: 1, Skills: map[int32]*Skill{1: {SkillId: 1, Level: 1}}},
	}
}


package cow

import "fmt"

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


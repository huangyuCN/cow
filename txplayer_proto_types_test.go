package cow

import "maps"

type Player struct {
	Name   string
	Level  int
	Items  map[int]int
	Skills []int
}

func newPlayer() *Player {
	return &Player{
		Name:   "hero",
		Level:  10,
		Items:  map[int]int{1001: 1, 1002: 2},
		Skills: []int{11, 22, 33},
	}
}

func newBenchPlayer(size int) *Player {
	items := make(map[int]int, size)
	skills := make([]int, size)
	for i := 0; i < size; i++ {
		items[i] = i
		skills[i] = i
	}
	return &Player{
		Name:   "hero",
		Level:  10,
		Items:  items,
		Skills: skills,
	}
}

func clonePlayer(src *Player) *Player {
	return &Player{
		Name:   src.Name,
		Level:  src.Level,
		Items:  maps.Clone(src.Items),
		Skills: append([]int(nil), src.Skills...),
	}
}

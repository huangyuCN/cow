package gamestore

import "github.com/google/go-cmp/cmp"

// clonePlayer 深拷贝 Player 快照（仅测试对比，夹具文件允许裸写构造副本）。
func clonePlayer(p *Player) *Player {
	if p == nil {
		return nil
	}
	c := &Player{
		Gold:     p.Gold,
		Wallet:   make(map[string]int64, len(p.Wallet)),
		Items:    make([]*Item, len(p.Items)),
		MainHero: cloneHero(p.MainHero),
		Heros:    make(map[int32]*Hero, len(p.Heros)),
		Bags:     make(map[int32][]*Item, len(p.Bags)),
		Stats:    make(map[int32]map[string]int64, len(p.Stats)),
	}
	for k, v := range p.Wallet {
		c.Wallet[k] = v
	}
	for i, it := range p.Items {
		if it != nil {
			c.Items[i] = &Item{Id: it.Id, Name: it.Name}
		}
	}
	for k, h := range p.Heros {
		c.Heros[k] = cloneHero(h)
	}
	for k, bag := range p.Bags {
		c.Bags[k] = cloneItems(bag)
	}
	for k, inner := range p.Stats {
		c.Stats[k] = make(map[string]int64, len(inner))
		for ik, iv := range inner {
			c.Stats[k][ik] = iv
		}
	}
	return c
}

func cloneHero(h *Hero) *Hero {
	if h == nil {
		return nil
	}
	c := &Hero{Level: h.Level, Skills: make(map[int32]*Skill, len(h.Skills))}
	for k, s := range h.Skills {
		if s != nil {
			c.Skills[k] = &Skill{Level: s.Level}
		}
	}
	return c
}

func cloneItems(s []*Item) []*Item {
	out := make([]*Item, len(s))
	for i, it := range s {
		if it != nil {
			out[i] = &Item{Id: it.Id, Name: it.Name}
		}
	}
	return out
}

func cloneGuild(g *Guild) *Guild {
	if g == nil {
		return nil
	}
	c := &Guild{Members: make(map[int32]*Member, len(g.Members))}
	for k, m := range g.Members {
		if m != nil {
			c.Members[k] = &Member{Name: m.Name, Rank: m.Rank}
		}
	}
	return c
}

func playersEqual(a, b *Player) bool {
	return cmp.Equal(clonePlayer(a), b)
}

func guildsEqual(a, b *Guild) bool {
	return cmp.Equal(cloneGuild(a), b)
}

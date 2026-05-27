package aftershop

import "github.com/google/go-cmp/cmp"

func NewDemoPlayer() *Player {
	return &Player{
		Gold:   1000,
		Wallet: map[string]int64{"gold": 1000},
		Items:  []*Item{{Id: 1, Name: "init"}},
		MainHero: &Hero{
			Level: 10,
		},
	}
}

func newDemoItem() *Item {
	return &Item{Id: 9, Name: "x"}
}

func clonePlayer(p *Player) *Player {
	if p == nil {
		return nil
	}
	out := &Player{
		Gold:     p.Gold,
		Wallet:   make(map[string]int64, len(p.Wallet)),
		Items:    make([]*Item, len(p.Items)),
		MainHero: cloneHero(p.MainHero),
	}
	for k, v := range p.Wallet {
		out.Wallet[k] = v
	}
	for i, it := range p.Items {
		if it == nil {
			continue
		}
		out.Items[i] = &Item{Id: it.Id, Name: it.Name}
	}
	return out
}

func cloneHero(h *Hero) *Hero {
	if h == nil {
		return nil
	}
	return &Hero{Level: h.Level}
}

func playersEqual(a, b *Player) bool {
	return cmp.Equal(clonePlayer(a), b)
}


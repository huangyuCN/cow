package legacy

import "github.com/huangyuCN/cow"

func Use(p *cow.Player, ctx *cow.TxContext) {
	p.Level = 1
	p.Assets["gold"] = 100
	if p.MainHero != nil {
		p.MainHero.Level = 2
	}
	p.Items = append(p.Items, &cow.Item{Id: 9, Name: "x"})
}

package consumer

func Use(p *Player, ctx *TxContext) {
	p.Level = 1
	p.Assets["gold"] = 100
	if p.MainHero != nil {
		p.MainHero.Level = 2
	}
	p.Items = append(p.Items, &Item{Id: 9})
}

package cow

// sparseWriteDirect 在副本上直接写（DeepCopy 对照组，无 Undo；仅 fixture 白名单）。
func sparseWriteDirect(p *Player) {
	p.Assets["gold"] = 500
	p.Items = append(p.Items, &Item{Id: 9999, Name: "Shield"})
	if p.MainHero != nil {
		p.MainHero.Level = 2
	}
}

// sparseWriteMegaDirect mega 档 DeepCopy 对照裸写。
func sparseWriteMegaDirect(p *Player) {
	p.Assets["gold"] = 500
	if h := p.Heros[1]; h != nil {
		h.Level = 99
	}
	p.Items = append(p.Items, &Item{Id: 9999, Name: "mega_probe"})
	p.Bags[1] = append(p.Bags[1], &Item{Id: 8888, Name: "bag_probe"})
	if inner := p.Stats[1]; inner != nil {
		inner["atk"] = 100
	}
}

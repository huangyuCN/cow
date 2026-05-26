package cow

// sparseWriteDirect 在副本上直接写（DeepCopy 对照组，无 Undo；仅 fixture 白名单）。
func sparseWriteDirect(p *Player) {
	p.Assets["gold"] = 500
	p.Items = append(p.Items, &Item{Id: 9999, Name: "Shield"})
	if p.MainHero != nil {
		p.MainHero.Level = 2
	}
}

// sparseWriteMegaDirect mega 档 DeepCopy 对照裸写（与 applyMegaSparseWrites 等价）。
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

// sparseWriteMegaDirect32 与 applyMegaSparseWrites32 等价的裸写（无 Undo）。
func sparseWriteMegaDirect32(p *Player) {
	p.Uid = 90001
	p.Level = 11
	p.Assets["gold"] = 500
	p.Assets["silver"] = 300
	p.Assets["gem"] = 50
	p.Items = append(p.Items, newTestItem(9999, "mega_probe"))
	if len(p.Items) > 0 {
		p.Items[0] = newTestItem(10000, "set0")
	}
	if len(p.Items) > 1 {
		p.Items = append(p.Items[:len(p.Items)-1])
	}
	if len(p.Items) > 2 {
		p.Items = p.Items[:2]
	}
	if p.MainHero != nil {
		p.MainHero.Level = 88
	}
	if h := p.Heros[1]; h != nil {
		h.Level = 99
	}
	p.Heros[99] = newTestHeroProbe99()
	p.Bags[1] = append(p.Bags[1], newTestItem(8888, "bag_probe"))
	p.Bags[2] = append(p.Bags[2], newTestItem(8887, "bag_probe2"))
	if len(p.Bags[1]) > 0 {
		p.Bags[1][0] = newTestItem(8886, "bagset")
	}
	p.Bags[3] = newTestItemsForBagPut()
	if inner := p.Stats[1]; inner != nil {
		inner["atk"] = 100
		inner["def"] = 80
	}
	if inner := p.Stats[2]; inner != nil {
		inner["hp"] = 200
	}
	if inner := p.Stats[3]; inner != nil {
		inner["mp"] = 150
	}
	p.Cooldowns[1] = append(p.Cooldowns[1], 100)
	if len(p.Cooldowns[1]) > 0 {
		p.Cooldowns[1][0] = 200
	}
	p.Cooldowns[2] = []int32{1, 2, 3}
	p.Cooldowns[3] = append(p.Cooldowns[3], 300)
	if m := p.Mails[1]; m != nil {
		m.Subject = "sub32"
	}
	p.Mails[2] = newTestMailPut()
	if q := p.Quests[1]; q != nil {
		q.State = 9
	}
	p.Quests[2] = newTestQuestPut()
}

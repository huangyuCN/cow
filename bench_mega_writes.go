package cow

// applyMegaSparseWrites 模拟一次请求的稀疏写（与 mega Benchmark 同源）。
func applyMegaSparseWrites(p *Player, ctx *TxContext) {
	p.PutAssets(ctx, "gold", 500)
	h := p.GetHeroForWrite(ctx, 1)
	if h != nil {
		h.PutLevel(ctx, 99)
	}
	p.AppendItems(ctx, &Item{Id: 9999, Name: "mega_probe"})
	p.AppendBagsAt(ctx, 1, &Item{Id: 8888, Name: "bag_probe"})
	p.PutStats(ctx, 1, "atk", 100)
}

// sparseWriteMegaDirect 在副本上直接写（DeepCopy 对照组，无 Undo）。
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

// applyMegaProxyProbeFull 对 Player 上各类生成代理逐项探针（单事务内顺序执行）。
func applyMegaProxyProbeFull(p *Player, ctx *TxContext) {
	// 标量
	p.PutUid(ctx, 90001)
	p.PutLevel(ctx, 42)
	// map 标量
	p.PutAssets(ctx, "probe_asset", 7)
	// 指针字段
	if mh := p.GetMainHeroForWrite(ctx); mh != nil {
		mh.PutLevel(ctx, 99)
	}
	// map[k]*Struct
	if h := p.GetHeroForWrite(ctx, 1); h != nil {
		h.PutLevel(ctx, 11)
	}
	p.PutHeros(ctx, 99, &Hero{
		HeroId: 99,
		Level:  5,
		Skills: map[int32]*Skill{1: {SkillId: 1, Level: 1}},
	})
	// []*Item 字段 slice
	p.AppendItems(ctx, &Item{Id: 70001, Name: "probe"})
	if len(p.Items) > 0 {
		p.SetItemsAt(ctx, 0, &Item{Id: 70002, Name: "set"})
	}
	if len(p.Items) > 1 {
		p.RemoveItemsAt(ctx, len(p.Items)-1)
	}
	if len(p.Items) > 2 {
		p.TruncateItems(ctx, 2)
	}
	// map[k][]*Item
	p.AppendBagsAt(ctx, 1, &Item{Id: 70003, Name: "bag"})
	if bag := p.Bags[1]; len(bag) > 0 {
		p.SetBagsAt(ctx, 1, 0, &Item{Id: 70004, Name: "bagset"})
		if it := p.GetItemAtForWrite(ctx, 1, 0); it != nil {
			it.PutName(ctx, "bag_item_probe")
		}
	}
	if len(p.Bags[1]) > 1 {
		p.RemoveBagsAt(ctx, 1, len(p.Bags[1])-1)
	}
	if len(p.Bags[2]) > 1 {
		p.TruncateBags(ctx, 2, 1)
	}
	p.PutBags(ctx, 3, []*Item{{Id: 70005, Name: "bag_put"}})
	// map[k]map[string]int64
	p.PutStats(ctx, 1, "probe_stat", 99)
	if inner := p.GetStatsMapForWrite(ctx, 2); inner != nil {
		inner["probe_inner"] = 1
	}
	// map[k][]int32
	p.AppendCooldownsAt(ctx, 1, 100)
	if cd := p.Cooldowns[1]; len(cd) > 0 {
		p.SetCooldownsAt(ctx, 1, 0, 200)
	}
	if len(p.Cooldowns[2]) > 1 {
		p.RemoveCooldownsAt(ctx, 2, len(p.Cooldowns[2])-1)
	}
	if len(p.Cooldowns[3]) > 1 {
		p.TruncateCooldowns(ctx, 3, 1)
	}
	p.PutCooldowns(ctx, 4, []int32{1, 2, 3})
	// map[k]*Mail / *Quest
	if m := p.GetMailForWrite(ctx, 1); m != nil {
		m.PutSubject(ctx, "probe_mail")
	}
	p.PutMails(ctx, 2, &Mail{Id: 2, Subject: "put_mail", Body: "body"})
	if q := p.GetQuestForWrite(ctx, 1); q != nil {
		q.PutState(ctx, 9)
	}
	p.PutQuests(ctx, 2, &Quest{Id: 2, State: 8, Objectives: map[int32]int32{0: 1}})
}

// applyMegaProxyProbe 保留别名，供 Rollback 全覆盖测试调用。
func applyMegaProxyProbe(p *Player, ctx *TxContext) {
	applyMegaProxyProbeFull(p, ctx)
}

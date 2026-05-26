package cow

// megaSparseWriteCount 为 32 档稀疏写目标 Undo 条数（与 BenchmarkMega_*SparseWrite32 同源）。
const megaSparseWriteCount = 32

// applyMegaSparseWrites 模拟一次请求的稀疏写（约 6 处 Undo，与历史 mega Benchmark 同源）。
func applyMegaSparseWrites(p *Player, ctx *TxContext) {
	p.PutAssets(ctx, "gold", 500)
	h := p.GetHeroForWrite(ctx, 1)
	if h != nil {
		h.PutLevel(ctx, 99)
	}
	p.AppendItems(ctx, newTestItem(9999, "mega_probe"))
	p.AppendBagsAt(ctx, 1, newTestItem(8888, "bag_probe"))
	p.PutStats(ctx, 1, "atk", 100)
}

// applyMegaSparseWrites32 在 Player 各根字段上均匀分布 32 处 Undo（每字段 2～4 处，避免堆叠在 Assets 等单字段）。
//
// 分布：Uid(1) Level(1) Assets(3) Items(4) MainHero(2) Heros(3) Bags(4) Stats(4) Cooldowns(4) Mails(3) Quests(3)
func applyMegaSparseWrites32(p *Player, ctx *TxContext) {
	// Uid、Level（标量）
	p.PutUid(ctx, 90001)
	p.PutLevel(ctx, 11)
	// Assets（map 标量，3 个键）
	p.PutAssets(ctx, "gold", 500)
	p.PutAssets(ctx, "silver", 300)
	p.PutAssets(ctx, "gem", 50)
	// Items（slice，4 种操作）
	p.AppendItems(ctx, newTestItem(9999, "mega_probe"))
	if len(p.Items) > 0 {
		p.SetItemsAt(ctx, 0, newTestItem(10000, "set0"))
	}
	if len(p.Items) > 1 {
		p.RemoveItemsAt(ctx, len(p.Items)-1)
	}
	if len(p.Items) > 2 {
		p.TruncateItems(ctx, 2)
	}
	// MainHero（指针）
	if mh := p.GetMainHeroForWrite(ctx); mh != nil {
		mh.PutLevel(ctx, 88)
	}
	// Heros（map 指针）
	if h := p.GetHeroForWrite(ctx, 1); h != nil {
		h.PutLevel(ctx, 99)
	}
	p.PutHeros(ctx, 99, newTestHeroProbe99())
	// Bags（map slice）
	p.AppendBagsAt(ctx, 1, newTestItem(8888, "bag_probe"))
	p.AppendBagsAt(ctx, 2, newTestItem(8887, "bag_probe2"))
	if len(p.Bags[1]) > 0 {
		p.SetBagsAt(ctx, 1, 0, newTestItem(8886, "bagset"))
	}
	p.PutBags(ctx, 3, newTestItemsForBagPut())
	// Stats（map map）
	p.PutStats(ctx, 1, "atk", 100)
	p.PutStats(ctx, 1, "def", 80)
	p.PutStats(ctx, 2, "hp", 200)
	p.PutStats(ctx, 3, "mp", 150)
	// Cooldowns（map slice）
	p.AppendCooldownsAt(ctx, 1, 100)
	if len(p.Cooldowns[1]) > 0 {
		p.SetCooldownsAt(ctx, 1, 0, 200)
	}
	p.PutCooldowns(ctx, 2, []int32{1, 2, 3})
	p.AppendCooldownsAt(ctx, 3, 300)
	// Mails（map 指针）
	if m := p.GetMailForWrite(ctx, 1); m != nil {
		m.PutSubject(ctx, "sub32")
	}
	p.PutMails(ctx, 2, newTestMailPut())
	// Quests（map 指针）
	if q := p.GetQuestForWrite(ctx, 1); q != nil {
		q.PutState(ctx, 9)
	}
	p.PutQuests(ctx, 2, newTestQuestPut())
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
	p.PutHeros(ctx, 99, newTestHeroProbe99())
	// []*Item 字段 slice
	p.AppendItems(ctx, newTestItem(70001, "probe"))
	if len(p.Items) > 0 {
		p.SetItemsAt(ctx, 0, newTestItem(70002, "set"))
	}
	if len(p.Items) > 1 {
		p.RemoveItemsAt(ctx, len(p.Items)-1)
	}
	if len(p.Items) > 2 {
		p.TruncateItems(ctx, 2)
	}
	// map[k][]*Item
	p.AppendBagsAt(ctx, 1, newTestItem(70003, "bag"))
	if bag := p.Bags[1]; len(bag) > 0 {
		p.SetBagsAt(ctx, 1, 0, newTestItem(70004, "bagset"))
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
	p.PutBags(ctx, 3, newTestItemsForBagPut())
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
	p.PutMails(ctx, 2, newTestMailPut())
	if q := p.GetQuestForWrite(ctx, 1); q != nil {
		q.PutState(ctx, 9)
	}
	p.PutQuests(ctx, 2, newTestQuestPut())
}

// applyMegaProxyProbe 保留别名，供 Rollback 全覆盖测试调用。
func applyMegaProxyProbe(p *Player, ctx *TxContext) {
	applyMegaProxyProbeFull(p, ctx)
}

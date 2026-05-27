package gamestore

import "errors"

// HandlePurchaseSuccess 演示成功提交：覆盖标量、map、slice、指针、map map、第二根 Guild。
func HandlePurchaseSuccess(p *Player, g *Guild, ctx *TxContext) error {
	p.PutGold(ctx, p.Gold-100)
	p.PutWallet(ctx, "gold", p.Wallet["gold"]+50)
	p.AppendItems(ctx, newLootItem())
	if len(p.Items) > 0 {
		p.SetItemsAt(ctx, 0, newUpgradedItem())
	}
	if mh := p.GetMainHeroForWrite(ctx); mh != nil {
		mh.PutLevel(ctx, mh.Level+1)
	}
	if h := p.GetHeroForWrite(ctx, 1); h != nil {
		h.PutLevel(ctx, h.Level+1)
	}
	p.PutStats(ctx, 1, "atk", 99)
	p.AppendBagsAt(ctx, 1, newBagDropItem())
	if inner := p.GetStatsMapForWrite(ctx, 1); inner != nil {
		inner["bonus"] = 1
	}
	g.PutMembers(ctx, 2, newMemberBob())
	return nil
}

// HandlePurchaseFail 中途失败，供 Rollback 演示。
func HandlePurchaseFail(p *Player, ctx *TxContext) error {
	p.PutGold(ctx, 99999)
	p.PutWallet(ctx, "gold", 0)
	return errors.New("payment declined")
}

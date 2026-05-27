package aftershop

import "errors"

// HandlePurchase 演示 cow 模式下的成功提交：所有写路径走 undoproxy-gen 代理方法。
func HandlePurchase(p *Player, ctx *TxContext) {
	p.PutGold(ctx, p.Gold-100)
	p.PutWallet(ctx, "gold", p.Wallet["gold"]+50)
	p.AppendItems(ctx, newDemoItem())
	if mh := p.GetMainHeroForWrite(ctx); mh != nil {
		mh.PutLevel(ctx, mh.Level+1)
	}
}

// HandlePurchaseFail 演示 cow 模式下的失败路径：在返回错误时由调用方 Rollback 恢复。
func HandlePurchaseFail(p *Player, ctx *TxContext) error {
	p.PutGold(ctx, 99999)
	p.PutWallet(ctx, "gold", 0)
	p.AppendItems(ctx, newDemoItem())
	if mh := p.GetMainHeroForWrite(ctx); mh != nil {
		mh.PutLevel(ctx, mh.Level+100)
	}
	return errors.New("payment declined")
}


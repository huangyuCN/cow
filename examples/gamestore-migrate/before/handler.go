package beforeshop

import "errors"

// HandlePurchase 演示裸写时代的成功提交：直接修改业务对象字段。
func HandlePurchase(p *Player) {
	p.Gold = p.Gold - 100
	p.Wallet["gold"] = p.Wallet["gold"] + 50
	p.Items = append(p.Items, newDemoItem())
	if p.MainHero != nil {
		p.MainHero.Level = p.MainHero.Level + 1
	}
}

// HandlePurchaseFail 演示裸写时代的失败路径：先产生写入，再返回错误，
// 以展示（在 after 迁移态）回滚能够恢复这些写入影响。
func HandlePurchaseFail(p *Player) error {
	p.Gold = 99999
	p.Wallet["gold"] = 0
	p.Items = append(p.Items, newDemoItem())
	if p.MainHero != nil {
		p.MainHero.Level = p.MainHero.Level + 100
	}
	return errors.New("payment declined")
}


package cow

import (
	"errors"
	"testing"
)

func TestMegaFixtureSize(t *testing.T) {
	p := newMegaBenchPlayer()
	got := approxPlayerHeapBytes(p)
	const want uint64 = 1 << 20
	lo := want * 85 / 100
	hi := want * 115 / 100
	if got < lo || got > hi {
		t.Fatalf("heap approx %d not in [%d,%d]", got, lo, hi)
	}
}

func TestMegaPlayer_BusinessPath_Rollback(t *testing.T) {
	p := newMegaBenchPlayer()
	want := clonePlayerSnapshot(p)
	err := runScopedWithRollback(p, func(ctx *TxContext) error {
		applyMegaSparseWrites(p, ctx)
		return errors.New("rollback")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertPlayerEqual(t, p, want)
}

func TestMegaPlayer_BusinessPath_Commit(t *testing.T) {
	p := newMegaBenchPlayer()
	err := runScopedCommit(p, func(ctx *TxContext) error {
		applyMegaSparseWrites(p, ctx)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assertMegaBusinessCommit(t, p)
}

func TestMegaPlayer_ProxyProbe_Rollback(t *testing.T) {
	p := newMegaBenchPlayer()
	want := clonePlayerSnapshot(p)
	err := runScopedWithRollback(p, func(ctx *TxContext) error {
		applyMegaProxyProbeFull(p, ctx)
		return errors.New("rollback")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	assertPlayerEqual(t, p, want)
}

func TestMegaPlayer_ProxyProbe_Commit(t *testing.T) {
	p := newMegaBenchPlayer()
	err := runScopedCommit(p, func(ctx *TxContext) error {
		applyMegaProxyProbeFull(p, ctx)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assertMegaProbeCommit(t, p)
}

// TestMegaPlayer_CommitPersistsAfterLaterRollback 提交后新事务 Rollback 不得撤销已提交状态。
func TestMegaPlayer_CommitPersistsAfterLaterRollback(t *testing.T) {
	p := newMegaBenchPlayer()
	if err := runScopedCommit(p, func(ctx *TxContext) error {
		p.PutLevel(ctx, 42)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ctx := txPool.Get().(*TxContext)
	ctx.Reset()
	defer txPool.Put(ctx)
	p.PutLevel(ctx, 99)
	ctx.Rollback()
	if p.Level != 42 {
		t.Fatalf("Level got %d want 42 after unrelated rollback", p.Level)
	}
}

// assertMegaBusinessCommit 校验业务短路径提交后的可见变更。
func assertMegaBusinessCommit(t *testing.T, p *Player) {
	t.Helper()
	if p.Assets["gold"] != 500 {
		t.Fatalf("gold got %d want 500", p.Assets["gold"])
	}
	if len(p.Items) == 0 || p.Items[len(p.Items)-1].Id != 9999 {
		t.Fatal("expected appended mega_probe item")
	}
	if h := p.Heros[1]; h == nil || h.Level != 99 {
		t.Fatalf("heros[1] level got %v want 99", h)
	}
	if len(p.Bags[1]) == 0 || p.Bags[1][len(p.Bags[1])-1].Id != 8888 {
		t.Fatal("expected bag_probe in bags[1]")
	}
	if v := p.Stats[1]["atk"]; v != 100 {
		t.Fatalf("stats[1][atk] got %d want 100", v)
	}
}

// assertMegaProbeCommit 校验全覆盖探针提交后各类型字段均已变更。
func assertMegaProbeCommit(t *testing.T, p *Player) {
	t.Helper()
	if p.Uid != 90001 {
		t.Fatalf("Uid got %d want 90001", p.Uid)
	}
	if p.Level != 42 {
		t.Fatalf("Level got %d want 42", p.Level)
	}
	if p.Assets["probe_asset"] != 7 {
		t.Fatalf("probe_asset got %d want 7", p.Assets["probe_asset"])
	}
	if p.MainHero == nil || p.MainHero.Level != 99 {
		t.Fatal("MainHero level not committed")
	}
	if h := p.Heros[1]; h == nil || h.Level != 11 {
		t.Fatal("Heros[1] level not committed")
	}
	if h99 := p.Heros[99]; h99 == nil || h99.Level != 5 {
		t.Fatal("PutHeros not committed")
	}
	if len(p.Items) < 2 || p.Items[0].Id != 70002 {
		t.Fatal("Items SetAt not committed")
	}
	if len(p.Items) != 2 {
		t.Fatalf("Items len got %d want 2 after remove+truncate", len(p.Items))
	}
	if bag0 := p.Bags[1][0]; bag0 == nil || bag0.Name != "bag_item_probe" {
		t.Fatal("GetItemAtForWrite bag not committed")
	}
	if p.Bags[3] == nil || len(p.Bags[3]) != 1 {
		t.Fatal("PutBags not committed")
	}
	if p.Stats[1]["probe_stat"] != 99 {
		t.Fatal("PutStats not committed")
	}
	if p.Stats[2]["probe_inner"] != 1 {
		t.Fatal("GetStatsMapForWrite inner write not committed")
	}
	if cd := p.Cooldowns[1]; len(cd) == 0 || cd[0] != 200 {
		t.Fatal("SetCooldownsAt not committed")
	}
	if p.Cooldowns[4] == nil || len(p.Cooldowns[4]) != 3 {
		t.Fatal("PutCooldowns not committed")
	}
	if m := p.Mails[1]; m == nil || m.Subject != "probe_mail" {
		t.Fatal("GetMailForWrite not committed")
	}
	if p.Mails[2] == nil || p.Mails[2].Subject != "put_mail" {
		t.Fatal("PutMails not committed")
	}
	if q := p.Quests[1]; q == nil || q.State != 9 {
		t.Fatal("GetQuestForWrite not committed")
	}
	if p.Quests[2] == nil || p.Quests[2].State != 8 {
		t.Fatal("PutQuests not committed")
	}
}

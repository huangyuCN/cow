package cow

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// clonePlayerSnapshot 用 deepcopy-gen 做测试间状态对比基线。
func clonePlayerSnapshot(p *Player) *Player {
	return p.DeepCopy()
}

// applySparseWrites 模拟一次请求的三处稀疏写（lite）。
func applySparseWrites(p *Player, ctx *TxContext) {
	p.PutAssets(ctx, "gold", 500)
	p.AppendItems(ctx, newTestItem(9999, "Shield"))
	h := p.GetMainHeroForWrite(ctx)
	if h != nil {
		h.PutLevel(ctx, 2)
	}
}

func assertPlayerEqual(t *testing.T, got, want *Player) {
	t.Helper()
	opts := []cmp.Option{
		cmp.Comparer(cmpItem),
		cmp.Comparer(cmpHero),
		cmp.Comparer(cmpSkill),
		cmp.Comparer(cmpMail),
		cmp.Comparer(cmpQuest),
	}
	if diff := cmp.Diff(want, got, opts...); diff != "" {
		t.Fatalf("player mismatch (-want +got):\n%s", diff)
	}
}

func cmpItem(a, b *Item) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Id == b.Id && a.Name == b.Name && a.Extra == b.Extra
}

func cmpSkill(a, b *Skill) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.SkillId == b.SkillId && a.Level == b.Level
}

func cmpHero(a, b *Hero) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.HeroId != b.HeroId || a.Level != b.Level {
		return false
	}
	if len(a.Skills) != len(b.Skills) {
		return false
	}
	for k, av := range a.Skills {
		bv, ok := b.Skills[k]
		if !ok || !cmpSkill(av, bv) {
			return false
		}
	}
	return true
}

func cmpMail(a, b *Mail) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Id == b.Id && a.Subject == b.Subject && a.Body == b.Body
}

func cmpQuest(a, b *Quest) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Id != b.Id || a.State != b.State {
		return false
	}
	if len(a.Objectives) != len(b.Objectives) {
		return false
	}
	for k, av := range a.Objectives {
		if b.Objectives[k] != av {
			return false
		}
	}
	return true
}

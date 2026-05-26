package cow

import "errors"

import "testing"

type v2EquivalenceCase struct {
	name    string
	applyV1 func(p *Player, ctx *TxContext)
	applyV2 func(p *Player, ctx *TxContextV2)
}

func v2EquivalenceCases() []v2EquivalenceCase {
	return []v2EquivalenceCase{
		{
			name: "sparse_writes",
			applyV1: func(p *Player, ctx *TxContext) {
				applySparseWrites(p, ctx)
			},
			applyV2: func(p *Player, ctx *TxContextV2) {
				applySparseWritesV2(p, ctx)
			},
		},
		{
			name: "repeated_writes",
			applyV1: func(p *Player, ctx *TxContext) {
				p.PutAssets(ctx, "gold", 111)
				p.PutAssets(ctx, "gold", 222)
				p.PutAssets(ctx, "token_new", 7)
				h := p.GetMainHeroForWrite(ctx)
				if h != nil {
					h.PutLevel(ctx, 3)
					h.PutLevel(ctx, 4)
				}
				p.AppendItems(ctx, newTestItem(80001, "A"))
				p.AppendItems(ctx, newTestItem(80002, "B"))
			},
			applyV2: func(p *Player, ctx *TxContextV2) {
				p.PutAssetsV2(ctx, "gold", 111)
				p.PutAssetsV2(ctx, "gold", 222)
				p.PutAssetsV2(ctx, "token_new", 7)
				h := p.GetMainHeroForWriteV2(ctx)
				if h != nil {
					h.PutLevelV2(ctx, 3)
					h.PutLevelV2(ctx, 4)
				}
				p.AppendItemsV2(ctx, newTestItem(80001, "A"))
				p.AppendItemsV2(ctx, newTestItem(80002, "B"))
			},
		},
		{
			name: "noop_and_append_mix",
			applyV1: func(p *Player, ctx *TxContext) {
				p.PutAssets(ctx, "gold", p.Assets["gold"])
				h := p.GetMainHeroForWrite(ctx)
				if h != nil {
					h.PutLevel(ctx, h.Level)
				}
				p.AppendItems(ctx, newTestItem(81001, "noop_mix"))
			},
			applyV2: func(p *Player, ctx *TxContextV2) {
				p.PutAssetsV2(ctx, "gold", p.Assets["gold"])
				h := p.GetMainHeroForWriteV2(ctx)
				if h != nil {
					h.PutLevelV2(ctx, h.Level)
				}
				p.AppendItemsV2(ctx, newTestItem(81001, "noop_mix"))
			},
		},
		{
			name: "remove_and_truncate_mix",
			applyV1: func(p *Player, ctx *TxContext) {
				p.AppendItems(ctx, newTestItem(82001, "r0"))
				p.AppendItems(ctx, newTestItem(82002, "r1"))
				if len(p.Items) > 3 {
					p.RemoveItemsAt(ctx, 1)
					p.TruncateItems(ctx, 3)
				}
			},
			applyV2: func(p *Player, ctx *TxContextV2) {
				p.AppendItemsV2(ctx, newTestItem(82001, "r0"))
				p.AppendItemsV2(ctx, newTestItem(82002, "r1"))
				if len(p.Items) > 3 {
					p.RemoveItemsAtV2(ctx, 1)
					p.TruncateItemsV2(ctx, 3)
				}
			},
		},
		{
			name: "bags_ops_mix",
			applyV1: func(p *Player, ctx *TxContext) {
				p.AppendBagsAt(ctx, 1, newTestItem(83001, "b0"))
				if len(p.Bags[1]) > 0 {
					p.SetBagsAt(ctx, 1, 0, newTestItem(83002, "b1"))
				}
				if len(p.Bags[1]) > 1 {
					p.RemoveBagsAt(ctx, 1, len(p.Bags[1])-1)
				}
				if len(p.Bags[1]) > 1 {
					p.TruncateBags(ctx, 1, 1)
				}
			},
			applyV2: func(p *Player, ctx *TxContextV2) {
				p.AppendBagsAtV2(ctx, 1, newTestItem(83001, "b0"))
				if len(p.Bags[1]) > 0 {
					p.SetBagsAtV2(ctx, 1, 0, newTestItem(83002, "b1"))
				}
				if len(p.Bags[1]) > 1 {
					p.RemoveBagsAtV2(ctx, 1, len(p.Bags[1])-1)
				}
				if len(p.Bags[1]) > 1 {
					p.TruncateBagsV2(ctx, 1, 1)
				}
			},
		},
		{
			name: "stats_cooldowns_mix",
			applyV1: func(p *Player, ctx *TxContext) {
				p.PutStats(ctx, 1, "atk", 777)
				inner := p.GetStatsMapForWrite(ctx, 2)
				inner["def"] = 333
				p.AppendCooldownsAt(ctx, 1, 99)
				if len(p.Cooldowns[1]) > 0 {
					p.SetCooldownsAt(ctx, 1, 0, 123)
				}
				if len(p.Cooldowns[2]) > 1 {
					p.RemoveCooldownsAt(ctx, 2, len(p.Cooldowns[2])-1)
				}
				if len(p.Cooldowns[3]) > 1 {
					p.TruncateCooldowns(ctx, 3, 1)
				}
			},
			applyV2: func(p *Player, ctx *TxContextV2) {
				p.PutStatsV2(ctx, 1, "atk", 777)
				inner := p.GetStatsMapForWriteV2(ctx, 2)
				inner["def"] = 333
				p.AppendCooldownsAtV2(ctx, 1, 99)
				if len(p.Cooldowns[1]) > 0 {
					p.SetCooldownsAtV2(ctx, 1, 0, 123)
				}
				if len(p.Cooldowns[2]) > 1 {
					p.RemoveCooldownsAtV2(ctx, 2, len(p.Cooldowns[2])-1)
				}
				if len(p.Cooldowns[3]) > 1 {
					p.TruncateCooldownsV2(ctx, 3, 1)
				}
			},
		},
		{
			name: "heros_ptr_mix",
			applyV1: func(p *Player, ctx *TxContext) {
				if h := p.GetHeroForWrite(ctx, 1); h != nil {
					h.PutLevel(ctx, 66)
				}
				p.PutHeros(ctx, 99, newTestHeroProbe99())
			},
			applyV2: func(p *Player, ctx *TxContextV2) {
				if h := p.GetHeroForWriteV2(ctx, 1); h != nil {
					h.PutLevelV2(ctx, 66)
				}
				p.PutHerosV2(ctx, 99, newTestHeroProbe99())
			},
		},
	}
}

// TestV2Equivalence_Commit 对比 V1/V2 在提交路径的最终状态是否一致。
func TestV2Equivalence_Commit(t *testing.T) {
	for _, tc := range v2EquivalenceCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			seed := newBenchPlayer()
			p1 := clonePlayerSnapshot(seed)
			p2 := clonePlayerSnapshot(seed)

			if err := runScopedCommit(p1, func(ctx *TxContext) error {
				tc.applyV1(p1, ctx)
				return nil
			}); err != nil {
				t.Fatalf("v1 commit error: %v", err)
			}
			if err := runScopedCommitV2(p2, func(ctx *TxContextV2) error {
				tc.applyV2(p2, ctx)
				return nil
			}); err != nil {
				t.Fatalf("v2 commit error: %v", err)
			}

			assertPlayerEqual(t, p1, p2)
		})
	}
}

// TestV2Equivalence_Rollback 对比 V1/V2 在回滚路径的最终状态是否一致且恢复到初始值。
func TestV2Equivalence_Rollback(t *testing.T) {
	for _, tc := range v2EquivalenceCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			seed := newBenchPlayer()
			p1 := clonePlayerSnapshot(seed)
			p2 := clonePlayerSnapshot(seed)

			errV1 := runScopedWithRollback(p1, func(ctx *TxContext) error {
				tc.applyV1(p1, ctx)
				return errors.New("rollback")
			})
			if errV1 == nil {
				t.Fatal("v1 expected rollback error")
			}
			errV2 := runScopedWithRollbackV2(p2, func(ctx *TxContextV2) error {
				tc.applyV2(p2, ctx)
				return errors.New("rollback")
			})
			if errV2 == nil {
				t.Fatal("v2 expected rollback error")
			}

			assertPlayerEqual(t, p1, p2)
			assertPlayerEqual(t, p1, seed)
		})
	}
}

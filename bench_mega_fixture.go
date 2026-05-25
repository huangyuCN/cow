package cow

import (
	"fmt"
	"strings"
)

// mega 夹具规模常量（调 TestMegaFixtureSize 直至 ~1MiB±15%）。
const (
	megaItemCount        = 2000
	megaItemNameLen      = 48
	megaHeroCount        = 120
	megaSkillsPerHero    = 40
	megaBagCount         = 30
	megaItemsPerBag      = 40
	megaMailCount        = 80
	megaMailBodyLen      = 4096
	megaQuestCount       = 100
	megaStatGroups       = 12
	megaStatKeysPerGroup = 50
	megaCooldownKeys     = 80
	megaCooldownListLen  = 24
	megaAssetCount       = 200
)

// newMegaBenchPlayer 构造堆上约 1MB 的游戏向 Player（用于探针与 mega Benchmark）。
func newMegaBenchPlayer() *Player {
	p := newBenchPlayer()
	p.Level = 10

	p.Assets = make(map[string]int64, megaAssetCount)
	p.Assets["gold"] = 1000
	for i := 0; i < megaAssetCount-1; i++ {
		p.Assets[fmt.Sprintf("asset_%d", i)] = int64(i + 1)
	}

	p.Items = make([]*Item, 0, megaItemCount)
	namePad := megaRepeat('i', megaItemNameLen)
	for i := 0; i < megaItemCount; i++ {
		p.Items = append(p.Items, &Item{
			Id:    int64(i + 1),
			Name:  namePad,
			Extra: megaRepeat('e', 16),
		})
	}

	p.Heros = make(map[int32]*Hero, megaHeroCount)
	for hid := int32(1); hid <= megaHeroCount; hid++ {
		skills := make(map[int32]*Skill, megaSkillsPerHero)
		for sid := int32(1); sid <= megaSkillsPerHero; sid++ {
			skills[sid] = &Skill{SkillId: sid, Level: int32(sid)}
		}
		p.Heros[hid] = &Hero{HeroId: hid, Level: 1, Skills: skills}
	}

	p.Bags = make(map[int32][]*Item, megaBagCount)
	for bid := int32(1); bid <= megaBagCount; bid++ {
		bag := make([]*Item, 0, megaItemsPerBag)
		for j := 0; j < megaItemsPerBag; j++ {
			bag = append(bag, &Item{Id: int64(bid)*1000 + int64(j), Name: namePad})
		}
		p.Bags[bid] = bag
	}

	p.Stats = make(map[int32]map[string]int64, megaStatGroups)
	for g := int32(1); g <= megaStatGroups; g++ {
		inner := make(map[string]int64, megaStatKeysPerGroup)
		for k := 0; k < megaStatKeysPerGroup; k++ {
			inner[fmt.Sprintf("stat_%d", k)] = int64(g)*1000 + int64(k)
		}
		p.Stats[g] = inner
	}

	p.Cooldowns = make(map[int32][]int32, megaCooldownKeys)
	for cid := int32(1); cid <= megaCooldownKeys; cid++ {
		cd := make([]int32, megaCooldownListLen)
		for i := range cd {
			cd[i] = int32(i + int(cid))
		}
		p.Cooldowns[cid] = cd
	}

	body := megaRepeat('m', megaMailBodyLen)
	p.Mails = make(map[uint64]*Mail, megaMailCount)
	for mid := uint64(1); mid <= megaMailCount; mid++ {
		p.Mails[mid] = &Mail{
			Id:      mid,
			Subject: fmt.Sprintf("mail_%d", mid),
			Body:    body,
		}
	}

	p.Quests = make(map[int32]*Quest, megaQuestCount)
	for qid := int32(1); qid <= megaQuestCount; qid++ {
		obj := make(map[int32]int32, 5)
		for o := int32(0); o < 5; o++ {
			obj[o] = qid*10 + o
		}
		p.Quests[qid] = &Quest{Id: qid, State: 1, Objectives: obj}
	}

	return p
}

func megaRepeat(ch byte, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(string(ch), n)
}

// approxPlayerHeapBytes 估算 Player 对象图占用（字节，允许 ±15% 误差）。
func approxPlayerHeapBytes(p *Player) uint64 {
	if p == nil {
		return 0
	}
	var n uint64
	n += 64 // 标量与 Player 壳
	n += approxMapStringInt64(p.Assets)
	n += approxSliceItems(p.Items)
	n += approxHero(p.MainHero)
	for _, h := range p.Heros {
		n += approxHero(h)
		n += 16
	}
	for _, bag := range p.Bags {
		n += approxSliceItems(bag)
		n += 24
	}
	for _, inner := range p.Stats {
		n += approxMapStringInt64(inner)
		n += 24
	}
	for _, cd := range p.Cooldowns {
		n += uint64(24 + 8*len(cd))
	}
	for _, m := range p.Mails {
		n += approxMail(m)
		n += 16
	}
	for _, q := range p.Quests {
		n += approxQuest(q)
		n += 16
	}
	return n
}

func approxMapStringInt64(m map[string]int64) uint64 {
	if m == nil {
		return 0
	}
	var n uint64 = 48
	for k := range m {
		n += uint64(16 + len(k) + 8)
	}
	return n
}

func approxSliceItems(s []*Item) uint64 {
	if s == nil {
		return 0
	}
	var n uint64 = 24
	for _, it := range s {
		n += approxItem(it) + 8
	}
	return n
}

func approxItem(it *Item) uint64 {
	if it == nil {
		return 0
	}
	return 32 + uint64(len(it.Name)+len(it.Extra))
}

func approxHero(h *Hero) uint64 {
	if h == nil {
		return 0
	}
	var n uint64 = 48
	for _, sk := range h.Skills {
		n += 32 + 16
		_ = sk
	}
	return n + uint64(8*len(h.Skills))
}

func approxMail(m *Mail) uint64 {
	if m == nil {
		return 0
	}
	return 48 + uint64(len(m.Subject)+len(m.Body))
}

func approxQuest(q *Quest) uint64 {
	if q == nil {
		return 0
	}
	return 48 + uint64(16+8*len(q.Objectives))
}


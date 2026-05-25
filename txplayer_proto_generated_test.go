package cow

import "maps"

type TxPlayer struct {
	base *Player

	name    string
	hasName bool

	level    int
	hasLevel bool

	items    map[int]int
	hasItems bool

	skills    []int
	hasSkills bool
}

func BeginPlayer(base *Player) *TxPlayer {
	return &TxPlayer{base: base}
}

func (tx *TxPlayer) Name() string {
	if tx.hasName {
		return tx.name
	}
	return tx.base.Name
}

func (tx *TxPlayer) SetName(v string) {
	tx.name = v
	tx.hasName = true
}

func (tx *TxPlayer) Level() int {
	if tx.hasLevel {
		return tx.level
	}
	return tx.base.Level
}

func (tx *TxPlayer) SetLevel(v int) {
	tx.level = v
	tx.hasLevel = true
}

func (tx *TxPlayer) ensureItems() map[int]int {
	if !tx.hasItems {
		tx.items = maps.Clone(tx.base.Items)
		tx.hasItems = true
	}
	return tx.items
}

func (tx *TxPlayer) Item(id int) (int, bool) {
	if tx.hasItems {
		v, ok := tx.items[id]
		return v, ok
	}
	v, ok := tx.base.Items[id]
	return v, ok
}

func (tx *TxPlayer) SetItem(id int, v int) {
	items := tx.ensureItems()
	items[id] = v
}

func (tx *TxPlayer) DeleteItem(id int) {
	items := tx.ensureItems()
	delete(items, id)
}

func (tx *TxPlayer) ensureSkills() []int {
	if !tx.hasSkills {
		tx.skills = append([]int(nil), tx.base.Skills...)
		tx.hasSkills = true
	}
	return tx.skills
}

func (tx *TxPlayer) Skill(i int) (int, bool) {
	var skills []int
	if tx.hasSkills {
		skills = tx.skills
	} else {
		skills = tx.base.Skills
	}
	if i < 0 || i >= len(skills) {
		return 0, false
	}
	return skills[i], true
}

func (tx *TxPlayer) SkillCount() int {
	if tx.hasSkills {
		return len(tx.skills)
	}
	return len(tx.base.Skills)
}

func (tx *TxPlayer) SetSkill(i int, v int) {
	skills := tx.ensureSkills()
	skills[i] = v
}

func (tx *TxPlayer) AppendSkill(v int) {
	skills := tx.ensureSkills()
	tx.skills = append(skills, v)
}

func (tx *TxPlayer) Commit() *Player {
	base := tx.base
	out := &Player{}

	if tx.hasName {
		out.Name = tx.name
	} else {
		out.Name = base.Name
	}

	if tx.hasLevel {
		out.Level = tx.level
	} else {
		out.Level = base.Level
	}

	if tx.hasItems {
		out.Items = tx.items
	} else {
		out.Items = base.Items
	}

	if tx.hasSkills {
		out.Skills = tx.skills
	} else {
		out.Skills = base.Skills
	}

	return out
}

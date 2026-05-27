package cow

// Item 模拟背包条目（标签兼容 PB/BSON；运行路径不依赖序列化）。
//
// +k8s:deepcopy-gen=true
type Item struct {
	Id    int64  `protobuf:"varint,1,opt,name=id" json:"id,omitempty" bson:"_id"`
	Name  string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty" bson:"name"`
	Extra string `json:"extra,omitempty" bson:"extra"`
}

// Hero 模拟英雄子结构。
//
// +k8s:deepcopy-gen=true
type Hero struct {
	HeroId int32             `protobuf:"varint,1,opt,name=hero_id" json:"hero_id,omitempty" bson:"hero_id"`
	Level  int32             `protobuf:"varint,2,opt,name=level" json:"level,omitempty" bson:"level"`
	Skills map[int32]*Skill  `json:"skills,omitempty" bson:"skills"`
}

// Player 模拟聚合根（lite + mega 双档构造）。
//
// +k8s:deepcopy-gen=true
// +cow:undoproxy-gen=true
type Player struct {
	Uid       int64                       `protobuf:"varint,1,opt,name=uid" json:"uid,omitempty" bson:"_id"`
	Level     int32                       `json:"level,omitempty" bson:"level"`
	Assets    map[string]int64            `protobuf:"bytes,2,rep,name=assets" json:"assets,omitempty" bson:"assets"`
	Items     []*Item                     `protobuf:"bytes,3,rep,name=items" json:"items,omitempty" bson:"items"`
	MainHero  *Hero                       `protobuf:"bytes,4,opt,name=hero" json:"hero,omitempty" bson:"hero"`
	Heros     map[int32]*Hero             `json:"heros,omitempty" bson:"heros"`
	Bags      map[int32][]*Item           `json:"bags,omitempty" bson:"bags"`
	Stats     map[int32]map[string]int64  `json:"stats,omitempty" bson:"stats"`
	Cooldowns map[int32][]int32           `json:"cooldowns,omitempty" bson:"cooldowns"`
	Mails     map[uint64]*Mail            `json:"mails,omitempty" bson:"mails"`
	Quests    map[int32]*Quest            `json:"quests,omitempty" bson:"quests"`
}

package cow

// Skill 技能子结构。
//
// +k8s:deepcopy-gen=true
type Skill struct {
	SkillId int32 `json:"skill_id,omitempty" bson:"skill_id"`
	Level   int32 `json:"level,omitempty" bson:"level"`
}

// Mail 邮件（Body 用于拉高 mega 夹具体积）。
//
// +k8s:deepcopy-gen=true
type Mail struct {
	Id      uint64 `json:"id,omitempty" bson:"id"`
	Subject string `json:"subject,omitempty" bson:"subject"`
	Body    string `json:"body,omitempty" bson:"body"`
}

// Quest 任务进度。
//
// +k8s:deepcopy-gen=true
type Quest struct {
	Id         int32           `json:"id,omitempty" bson:"id"`
	State      int32           `json:"state,omitempty" bson:"state"`
	Objectives map[int32]int32 `json:"objectives,omitempty" bson:"objectives"`
}

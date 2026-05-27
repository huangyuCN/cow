package domain

// Player 聚合根（测试用）。
//
// +cow:undoproxy-gen=true
type Player struct {
	Hunt *BossHunt
}

// BossHunt 纳入本包 undoproxy 类型图。
type BossHunt struct {
	CurrLevel int32
	CurrPass  int32
	ClaimPass int32
}

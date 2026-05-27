package consumer

// TxContext 测试用最小上下文（真实接入由生成器产出）。
type TxContext struct{}

type Hero struct {
	Level int32
}

type Item struct {
	Id int64
}

// Player 带 undoproxy 标记的聚合根。
//
// +cow:undoproxy-gen=true
type Player struct {
	Level    int32
	Assets   map[string]int64
	MainHero *Hero
	Items    []*Item
}

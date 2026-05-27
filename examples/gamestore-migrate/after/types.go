package aftershop

// +cow:undoproxy-gen=true
type Player struct {
	Gold     int64
	Wallet   map[string]int64
	Items    []*Item
	MainHero *Hero
}

type Item struct {
	Id   int64
	Name string
}

type Hero struct {
	Level int32
}


package beforeshop

func NewDemoPlayer() *Player {
	return &Player{
		Gold:   1000,
		Wallet: map[string]int64{"gold": 1000},
		Items:  []*Item{{Id: 1, Name: "init"}},
		MainHero: &Hero{
			Level: 10,
		},
	}
}

func newDemoItem() *Item {
	return &Item{Id: 9, Name: "x"}
}


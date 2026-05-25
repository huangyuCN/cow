package barewrite

func BadLiteral() *Player {
	return &Player{Level: 1} // want `cowbarewrite:`
}

package barewrite

func BadDelete(p *Player) {
	delete(p.Heros, 1) // want `cowbarewrite:`
}

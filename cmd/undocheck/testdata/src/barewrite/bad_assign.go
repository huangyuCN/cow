package barewrite

func BadAssign(p *Player) {
	p.Level = 1 // want `cowbarewrite:`
}

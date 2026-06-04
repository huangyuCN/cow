package consumer

import "importscope/defpkg"

func BadAssign(p *defpkg.Player) {
	p.Level = 1 // want `cowbarewrite:`
}

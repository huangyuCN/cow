package consumer

import "importscope/defpkg"

func GoodRead(p *defpkg.Player) int32 {
	return p.Level
}

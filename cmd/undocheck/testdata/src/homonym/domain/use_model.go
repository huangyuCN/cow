package domain

import "homonym/model"

// 对外部包同名 struct 裸写不应误报。
func useModel(h *model.BossHunt) {
	h.CurrLevel = 1
	h.CurrPass = 2
	h.ClaimPass = 3
}

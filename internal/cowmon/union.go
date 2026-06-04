package cowmon

import "go/types"

// Union 合并多个监控集；无有效输入时返回 nil。
func Union(sets ...*MonitoredSet) *MonitoredSet {
	var out *MonitoredSet
	for _, s := range sets {
		if s == nil || len(s.byObj) == 0 {
			continue
		}
		if out == nil {
			out = &MonitoredSet{byObj: make(map[*types.TypeName]struct{})}
		}
		for obj := range s.byObj {
			out.byObj[obj] = struct{}{}
		}
	}
	if out == nil {
		return nil
	}
	return out
}

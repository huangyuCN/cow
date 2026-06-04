package cowmon

import "sync"

var monitoredCache sync.Map // import path -> *monitoredCacheEntry

type monitoredCacheEntry struct {
	set *MonitoredSet
	err error
}

// LoadMonitoredCached 按 import 路径加载监控集，进程内缓存成功与失败结果。
// srcDir 非空时在 module 加载失败后回退为 GOPATH 式 testdata/src 加载（供 analysistest 使用）。
func LoadMonitoredCached(importPath, srcDir string) (*MonitoredSet, error) {
	key := cacheKey(importPath, srcDir)
	if v, ok := monitoredCache.Load(key); ok {
		e := v.(*monitoredCacheEntry)
		return e.set, e.err
	}
	set, err := LoadMonitored(importPath)
	if err != nil && srcDir != "" {
		set, err = LoadMonitoredFromSrcDir(importPath, srcDir)
	}
	monitoredCache.Store(key, &monitoredCacheEntry{set: set, err: err})
	return set, err
}

func cacheKey(importPath, srcDir string) string {
	if srcDir == "" {
		return importPath
	}
	return importPath + "\x00" + srcDir
}

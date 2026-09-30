// Package bootstrap 资源释放注册表：模块显式初始化成功后注册释放函数，
// 停机时按注册逆序执行（数据库、Redis 初始化时自动注册）。
package bootstrap

import "sync"

var (
	mu                 sync.RWMutex
	registeredReleases = map[string]func(){} // 已注册的释放函数（名称 -> 函数）
	releaseOrder       []string              // 释放函数注册顺序，释放时按逆序执行
)

// 注册资源释放函数，服务停止时按注册的逆序执行（数据库、Redis 显式初始化时自动注册）。
// 入参: name (模块名称), fn (释放函数)
func RegisterRelease(name string, fn func()) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registeredReleases[name]; !exists {
		releaseOrder = append(releaseOrder, name)
	}
	registeredReleases[name] = fn
}

// 按注册逆序执行全部资源释放函数。
// 实现为「加锁取快照 -> 释放锁 -> 执行回调」：回调内再次调用 RegisterRelease 不会死锁，
// 且新注册的释放函数不参与本轮执行。
func Release() {
	mu.RLock()
	names := make([]string, len(releaseOrder))
	copy(names, releaseOrder)
	releases := make(map[string]func(), len(registeredReleases))
	for name, fn := range registeredReleases {
		releases[name] = fn
	}
	mu.RUnlock()
	for i := len(names) - 1; i >= 0; i-- {
		if fn, ok := releases[names[i]]; ok {
			fn()
		}
	}
}

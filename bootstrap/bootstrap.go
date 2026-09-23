package bootstrap

import (
	"sync"

	"gopkg.761sama.com/tenon/conf"
)

// InitFunc 初始化模块函数类型，接收总配置。
type InitFunc func(cfg conf.Config)

var (
	mu                     sync.RWMutex
	RegisteredInitModules  = map[string]InitFunc{} // 已注册的初始化模块（名称 -> 函数）
	registerOrder          []string                // 模块注册顺序，保证初始化按注册先后执行
	registeredReleases     = map[string]func(){}
	releaseOrder           []string
)

// 注册初始化模块，同名重复注册会覆盖并保留原顺序。
// 入参: name (模块名称), fn (初始化函数)
func RegisterInitModule(name string, fn InitFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := RegisteredInitModules[name]; !exists {
		registerOrder = append(registerOrder, name)
	}
	RegisteredInitModules[name] = fn
}

// 注册资源释放函数，服务停止时按注册的逆序执行。
// 入参: name (模块名称), fn (释放函数)
func RegisterRelease(name string, fn func()) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registeredReleases[name]; !exists {
		releaseOrder = append(releaseOrder, name)
	}
	registeredReleases[name] = fn
}

// 按注册顺序执行全部初始化模块。
// 入参: cfg (总配置)
func Init(cfg conf.Config) {
	mu.RLock()
	defer mu.RUnlock()
	for _, name := range registerOrder {
		RegisteredInitModules[name](cfg)
	}
}

// 执行单个初始化模块，模块不存在时返回 false。
// 入参: name (模块名称), cfg (总配置)
// 出参: 模块是否存在并已执行
func TinyInit(name string, cfg conf.Config) bool {
	mu.RLock()
	defer mu.RUnlock()
	fn, ok := RegisteredInitModules[name]
	if !ok {
		return false
	}
	fn(cfg)
	return true
}

// 按注册逆序执行全部资源释放函数。
func Release() {
	mu.RLock()
	defer mu.RUnlock()
	for i := len(releaseOrder) - 1; i >= 0; i-- {
		registeredReleases[releaseOrder[i]]()
	}
}

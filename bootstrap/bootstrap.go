package bootstrap

import (
	"fmt"
	"sync"
)

// InitFunc 初始化模块函数类型：可变参数，失败返回错误由调用方决定处理方式（如 Fatal）。
type InitFunc func(args ...string) error

var (
	mu                 sync.RWMutex
	RegisteredInitModules = map[string]InitFunc{} // 已注册的初始化模块（名称 -> 函数）
	registerOrder      []string                   // 模块注册顺序，保证初始化按注册先后执行
	registeredReleases = map[string]func(){}
	releaseOrder       []string
)

// 注册初始化模块（使用方扩展点），同名重复注册会覆盖并保留原顺序。
// 入参: name (模块名称), fn (初始化函数)
func RegisterInitModule(name string, fn InitFunc) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := RegisteredInitModules[name]; !exists {
		registerOrder = append(registerOrder, name)
	}
	RegisteredInitModules[name] = fn
}

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

// 按注册顺序执行全部初始化模块，任一模块失败即中断并返回错误。
// 入参: args (透传给各初始化函数的参数)
// 出参: 首个失败模块的错误
func Init(args ...string) error {
	mu.RLock()
	defer mu.RUnlock()
	for _, name := range registerOrder {
		if err := RegisteredInitModules[name](args...); err != nil {
			return fmt.Errorf("init module %q failed: %w", name, err)
		}
	}
	return nil
}

// 执行单个初始化模块。
// 入参: name (模块名称), args (透传给初始化函数的参数)
// 出参: 模块不存在或执行失败时返回错误
func TinyInit(name string, args ...string) error {
	mu.RLock()
	defer mu.RUnlock()
	fn, ok := RegisteredInitModules[name]
	if !ok {
		return fmt.Errorf("init module not found: %s", name)
	}
	return fn(args...)
}

// 按注册逆序执行全部资源释放函数。
func Release() {
	mu.RLock()
	defer mu.RUnlock()
	for i := len(releaseOrder) - 1; i >= 0; i-- {
		registeredReleases[releaseOrder[i]]()
	}
}

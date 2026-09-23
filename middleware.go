package tenon

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	mwMu       sync.RWMutex
	middlewares = map[string]gin.HandlerFunc{} // 已注册的中间件（名称 -> 函数）
)

// 注册中间件，注册后可通过 Middleware("名称") 获取并传入 Router。
// 入参: name (中间件名称), mw (中间件函数)
func RegMiddleware(name string, mw gin.HandlerFunc) {
	if mw == nil {
		panic(fmt.Sprintf("tenon: middleware %q is nil", name))
	}
	mwMu.Lock()
	defer mwMu.Unlock()
	middlewares[name] = mw
}

// 按名称获取已注册的中间件函数，未注册时 panic（启动期编程错误应尽早暴露）。
// 入参: name (中间件名称)
// 出参: 中间件函数
func Middleware(name string) gin.HandlerFunc {
	mwMu.RLock()
	defer mwMu.RUnlock()
	mw, ok := middlewares[name]
	if !ok {
		panic(fmt.Sprintf("tenon: middleware %q not registered", name))
	}
	return mw
}

// 判断中间件是否已注册。
// 入参: name (中间件名称)
// 出参: 是否已注册
func HasMiddleware(name string) bool {
	mwMu.RLock()
	defer mwMu.RUnlock()
	_, ok := middlewares[name]
	return ok
}

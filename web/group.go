package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RouterGroup 路由组，共享前缀与组级中间件。
type RouterGroup struct {
	group *gin.RouterGroup
}

// 在路由组上注册路由：前若干参数为中间件函数，最后一个参数为控制器。
// 入参: method (HTTP 方法), path (相对路由路径), handlers (中间件函数与控制器的有序列表)
func (g *RouterGroup) Router(method, path string, handlers ...gin.HandlerFunc) {
	handle(g.group, method, path, handlers)
}

// 注册组级中间件。必须在组内注册任何路由之前调用（gin 的 Use 在已有路由注册后调用会 panic）。
// 入参: mw (中间件函数列表)
func (g *RouterGroup) Use(mw ...gin.HandlerFunc) {
	g.group.Use(mw...)
}

// 在路由组上注册 GET 路由。
// 入参: path (相对路由路径), handlers (中间件函数与控制器的有序列表)
func (g *RouterGroup) GET(path string, handlers ...gin.HandlerFunc) {
	g.Router(http.MethodGet, path, handlers...)
}

// 在路由组上注册 POST 路由。
// 入参: path (相对路由路径), handlers (中间件函数与控制器的有序列表)
func (g *RouterGroup) POST(path string, handlers ...gin.HandlerFunc) {
	g.Router(http.MethodPost, path, handlers...)
}

// 在路由组上注册 PUT 路由。
// 入参: path (相对路由路径), handlers (中间件函数与控制器的有序列表)
func (g *RouterGroup) PUT(path string, handlers ...gin.HandlerFunc) {
	g.Router(http.MethodPut, path, handlers...)
}

// 在路由组上注册 DELETE 路由。
// 入参: path (相对路由路径), handlers (中间件函数与控制器的有序列表)
func (g *RouterGroup) DELETE(path string, handlers ...gin.HandlerFunc) {
	g.Router(http.MethodDelete, path, handlers...)
}

// 在路由组上注册 PATCH 路由。
// 入参: path (相对路由路径), handlers (中间件函数与控制器的有序列表)
func (g *RouterGroup) PATCH(path string, handlers ...gin.HandlerFunc) {
	g.Router(http.MethodPatch, path, handlers...)
}

// 在路由组上注册 HEAD 路由。
// 入参: path (相对路由路径), handlers (中间件函数与控制器的有序列表)
func (g *RouterGroup) HEAD(path string, handlers ...gin.HandlerFunc) {
	g.Router(http.MethodHead, path, handlers...)
}

// 在路由组上注册 OPTIONS 路由。
// 入参: path (相对路由路径), handlers (中间件函数与控制器的有序列表)
func (g *RouterGroup) OPTIONS(path string, handlers ...gin.HandlerFunc) {
	g.Router(http.MethodOptions, path, handlers...)
}

// 在路由组上注册匹配所有 HTTP 方法的路由。
// 入参: path (相对路由路径), handlers (中间件函数与控制器的有序列表)
func (g *RouterGroup) ANY(path string, handlers ...gin.HandlerFunc) {
	g.Router("ANY", path, handlers...)
}

// 创建嵌套路由组。
// 入参: prefix (相对路由前缀), handlers (组级中间件函数列表)
// 出参: 嵌套路由组
func (g *RouterGroup) Group(prefix string, handlers ...gin.HandlerFunc) *RouterGroup {
	return &RouterGroup{group: g.group.Group(prefix, handlers...)}
}

// 获取底层 gin 路由组。
// 出参: gin 路由组
func (g *RouterGroup) Raw() *gin.RouterGroup {
	return g.group
}

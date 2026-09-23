package web

import (
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

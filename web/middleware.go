package web

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"gopkg.761sama.com/tenon/common"
	"gopkg.761sama.com/tenon/conf"
)

// 全局异常恢复中间件：捕获 panic 并返回 500 错误。
// 出参: gin 中间件函数
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf("panic recovered: %+v\n%s", err, debug.Stack())
				common.ErrorResponse(c, "服务器内部错误")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// 跨域资源共享 (CORS) 中间件：AllowOrigins 为空时关闭跨域（不附加任何 CORS 头，由浏览器默认拦截）；
// 显式配置来源列表（含 ["*"]）时启用。
// 入参: cfg (HTTP 配置)
// 出参: gin 中间件函数
func CorsMiddleware(cfg conf.HTTPConfig) gin.HandlerFunc {
	if len(cfg.AllowOrigins) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	config := cors.DefaultConfig()
	config.AllowOrigins = cfg.AllowOrigins
	if len(cfg.AllowMethods) > 0 {
		config.AllowMethods = cfg.AllowMethods
	}
	if len(cfg.AllowHeaders) > 0 {
		config.AllowHeaders = cfg.AllowHeaders
	}
	return cors.New(config)
}

// 请求体大小限制中间件：Content-Length 超限直接 413；
// 同时以 MaxBytesReader 包裹请求体，覆盖 chunked 传输场景。
// 入参: max (最大字节数，负数表示不限制，0 使用默认值 32MB)
// 出参: gin 中间件函数
func MaxBodySizeMiddleware(max int64) gin.HandlerFunc {
	if max == 0 {
		max = 32 << 20
	}
	return func(c *gin.Context) {
		if max < 0 {
			c.Next()
			return
		}
		if c.Request.ContentLength > max {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		}
		c.Next()
	}
}

// 全局无路由处理。
// 入参: c (gin 上下文)
func NoRouteHandle(c *gin.Context) {
	common.ErrorResponse(c, "API 异常")
}

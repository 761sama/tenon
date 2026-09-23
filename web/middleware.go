package web

import (
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

// 跨域资源共享 (CORS) 中间件。
// 入参: cfg (HTTP 配置)
// 出参: gin 中间件函数
func CorsMiddleware(cfg conf.HTTPConfig) gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowOrigins = cfg.AllowOrigins
	config.AllowHeaders = cfg.AllowHeaders
	config.AllowMethods = cfg.AllowMethods
	return cors.New(config)
}

// 全局无路由处理。
// 入参: c (gin 上下文)
func NoRouteHandle(c *gin.Context) {
	common.ErrorResponse(c, "API 异常")
}

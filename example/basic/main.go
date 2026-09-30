package main

import (
	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon"
	"gopkg.761sama.com/tenon/web"
)

// 注册示例中间件：为响应添加标记头。
func markMiddleware(c *gin.Context) {
	c.Header("X-Powered-By", "tenon")
	c.Next()
}

// 手工装配示例：不使用 WebApp，逐个显式初始化日志、构建 Web 服务，展示框架的自由度。
func main() {
	tenon.RegMiddleware("mark", markMiddleware)
	if err := tenon.InitLog(tenon.LogConfig{Enable: false}, true); err != nil {
		panic(err)
	}
	cfg := tenon.DefaultHTTPConfig()
	cfg.Debug = true
	cfg.Address = "127.0.0.1"
	cfg.Port = 8080
	server, err := web.New(cfg)
	if err != nil {
		panic(err)
	}
	server.Router("GET", "/ping", tenon.Middleware("mark"), func(c *gin.Context) {
		tenon.Success(c, gin.H{"msg": "pong"})
	})
	v1 := server.Group("/v1")
	v1.Router("GET", "/hello", func(c *gin.Context) {
		tenon.Success(c, "hello tenon")
	})
	if err := server.Run(); err != nil {
		panic(err)
	}
}

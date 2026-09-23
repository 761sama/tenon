package main

import (
	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon"
)

// 注册示例中间件：为响应添加标记头。
func markMiddleware(c *gin.Context) {
	c.Header("X-Powered-By", "tenon")
	c.Next()
}

func main() {
	tenon.RegMiddleware("mark", markMiddleware)
	cfg := tenon.DefaultHTTPConfig()
	cfg.Debug = true
	cfg.Address = "127.0.0.1"
	cfg.Port = 8080
	server := tenon.WebServer(cfg)
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

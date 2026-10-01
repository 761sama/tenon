package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon"
	"gopkg.761sama.com/tenon/app"
	"gopkg.761sama.com/tenon/web"
)

// echoForm echo 接口的请求参数。
type echoForm struct {
	Name string `json:"name" form:"name"`
}

func main() {
	cfg := tenon.DefaultHTTPConfig()
	cfg.Address = "127.0.0.1"
	cfg.Port = 8083
	startedAt := time.Now()
	application := tenon.NewWebApp(tenon.WebAppOptions{
		HTTP: cfg,
		Log:  tenon.LogConfig{Enable: false},
		// 主服务（HTTP）无需声明，恒定启动；Enable 只列可选模块，顺序即初始化顺序。
		Enable: []string{
			tenon.ModuleLog,
			tenon.ModuleRequestID,
			"uptime", // 自定义模块经 Custom 注册后按名启用
		},
		Custom: []app.Module{{
			Name: "uptime",
			Init: func() error {
				startedAt = time.Now()
				fmt.Println("custom module [uptime] initialized")
				return nil
			},
			Release: func() {
				fmt.Println("custom module [uptime] released")
			},
		}},
	})
	s := application.Server()
	s.GET("/ping", func(c *gin.Context) {
		tenon.Success(c, gin.H{"msg": "pong", "request_id": c.GetString(web.RequestIDKey)})
	})
	s.GET("/uptime", func(c *gin.Context) {
		tenon.Success(c, gin.H{"uptime": time.Since(startedAt).String()})
	})
	s.POST("/echo", func(c *gin.Context) {
		var f echoForm
		if err := tenon.Bind(c, &f); err != nil {
			tenon.Error(c, "参数绑定失败: "+err.Error())
			return
		}
		tenon.Success(c, gin.H{"name": f.Name})
	})
	if err := application.Run(); err != nil {
		panic(err)
	}
}

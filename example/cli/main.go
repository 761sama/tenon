package main

import (
	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon"
)

func main() {
	cfg := tenon.DefaultHTTPConfig()
	cfg.Address = "127.0.0.1"
	cfg.Port = 8081
	server := tenon.WebServer(cfg)
	server.Router("GET", "/ping", func(c *gin.Context) {
		tenon.Success(c, gin.H{"msg": "pong"})
	})
	cli := tenon.NewCli("demo", "tenon CLI 示例")
	cli.AddCommand("version", "打印版本", func() {
		println("tenon " + tenon.Version)
	})
	cli.Run(server)
}

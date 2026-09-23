package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

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
	// 带 flags 与嵌套子命令的命令组示例
	cli.Add(tenon.Command{
		Use:   "limit",
		Short: "限制管理",
		Sub: []tenon.Command{
			{
				Use:   "list",
				Short: "列出限制",
				Flags: []tenon.CmdFlag{{Name: "page", Shorthand: "p", Usage: "页码", Default: 1}},
				Run: func(cmd *cobra.Command, args []string) {
					page, _ := cmd.Flags().GetInt("page")
					fmt.Printf("limit list, page=%d\n", page)
				},
			},
			{
				Use:   "unlock <ip>",
				Short: "解除 IP 限制",
				Args:  cobra.ExactArgs(1),
				Flags: []tenon.CmdFlag{{Name: "force", Shorthand: "f", Usage: "强制解除", Default: false}},
				Run: func(cmd *cobra.Command, args []string) {
					force, _ := cmd.Flags().GetBool("force")
					fmt.Printf("unlock ip=%s force=%v\n", args[0], force)
				},
			},
		},
	})
	cli.Run(server)
}

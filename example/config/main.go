package main

import (
	"fmt"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon"
)

const appVersion = "1.0.0"

// AppConfig 应用配置：组合 tenon 的配置结构，与框架默认配置解耦。
type AppConfig struct {
	Version string               `json:"version"`
	HTTP    tenon.HTTPConfig     `json:"http"`
	Log     tenon.LogConfig      `json:"log"`
	DB      tenon.DatabaseConfig `json:"db"`
}

func (c *AppConfig) SetVersion(v string) { c.Version = v }

func defaultAppConfig() *AppConfig {
	cfg := &AppConfig{HTTP: tenon.DefaultHTTPConfig()}
	cfg.HTTP.Address = "127.0.0.1"
	cfg.HTTP.Port = 8082
	// 默认使用 sqlite 本地文件，开箱即可运行
	cfg.DB.Type = "sqlite3"
	cfg.DB.Master.DBFile = filepath.Join("data", "app.db")
	return cfg
}

func main() {
	cli := tenon.NewCli("app", "配置驱动示例")
	flags := cli.BindFlags()
	cli.AddCommand("server", "启动服务", func() {
		configPath := flags.ConfigPath
		if configPath == "" {
			configPath = filepath.Join(flags.DataDir, "config.json")
		}
		cfg, from, err := tenon.LoadConfig(configPath, defaultAppConfig, appVersion)
		if err != nil {
			panic(err)
		}
		fmt.Printf("config %s: %s\n", from, configPath)
		cfg.HTTP.Debug = flags.Debug
		// 主服务（HTTP）恒定启动，Enable 只列可选模块；未列出的模块完全不参与
		application := tenon.NewWebApp(tenon.WebAppOptions{
			HTTP: cfg.HTTP, Log: cfg.Log, Database: cfg.DB,
			Enable: []string{tenon.ModuleLog, tenon.ModuleDatabase},
		})
		s := application.Server()
		s.Router("GET", "/ping", func(c *gin.Context) {
			tenon.Success(c, gin.H{"msg": "pong", "version": cfg.Version})
		})
		if err := application.Run(); err != nil {
			panic(err)
		}
	})
	cli.Execute()
}

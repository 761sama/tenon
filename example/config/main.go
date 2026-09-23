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
	Version string           `json:"version"`
	HTTP    tenon.HTTPConfig `json:"http"`
}

func (c *AppConfig) SetVersion(v string) { c.Version = v }

func defaultAppConfig() *AppConfig {
	cfg := &AppConfig{HTTP: tenon.DefaultHTTPConfig()}
	cfg.HTTP.Address = "127.0.0.1"
	cfg.HTTP.Port = 8082
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
		server := tenon.WebServer(cfg.HTTP)
		server.Router("GET", "/ping", func(c *gin.Context) {
			tenon.Success(c, gin.H{"msg": "pong", "version": cfg.Version})
		})
		if err := server.Run(); err != nil {
			panic(err)
		}
	})
	cli.Execute()
}

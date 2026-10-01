# tenon

> 通用 Go 服务框架：注册制设计，装配层编排生命周期，当前支持 Web 服务。

## 简介

tenon 是一个通用服务框架库，将 Web 服务、数据库、缓存、日志等基础设施以注册制方式封装，
使用方只需关注业务分层（router - controller - handler - model - dao），通过 `tenon.xxx` 快捷调用即可搭起服务骨架。

0.2 起提供预制装配层：`tenon.NewWebApp` 一行装配应用——主服务（HTTP）恒定启动，
可选模块（日志 / 数据库 / Redis / 请求 ID）按 `Enable` 清单选择性加载，初始化失败自动回滚，停机自动逆序释放。

## 项目信息

| 项目 | 内容 |
|------|------|
| 项目类型 | 库（Go 框架） |
| 项目定位 | 通用 Go 服务框架，当前支持 Web 服务 |
| 主要语言 / 版本 | Go 1.26 |
| 构建 / 包管理工具 | go build / go mod |
| 测试框架 | go test |
| 测试目录约定 | 测试与源码同目录（`*_test.go`） |
| 代码注释语言 | 中文 |
| 提交信息规范 | 统一固定格式，见 dev.md 第 2.4 节 |
| 特殊约定 | 注册制设计（中间件、模型、模块均可插拔注册）；装配层编排生命周期；允许引入成熟第三方依赖 |

## 功能特性

- 预制 Web 应用：`tenon.NewWebApp` 装配——主服务（HTTP/HTTPS/HTTP3(QUIC)）恒定启动，可选模块按 `Enable` 清单加载，初始化失败自动逆序回滚
- 装配层 `app/`：模块注册（`Add`）与启用（`Enable`）分离，自定义模块经 `Custom` 注册后按名启用；停机按初始化逆序释放
- Web 服务：基于 gin，路由方法糖（`GET`/`POST`/.../`ANY`）、CORS、内置安全响应头、HTTPS 最低 TLS 版本可配、优雅停机、路由组
- 中间件注册制：`tenon.RegMiddleware` 注册即返回句柄，`tenon.Middleware` 按名取用（配置驱动场景）
- 数据库模块：gorm 封装，支持 sqlite3 / mysql（可选 TLS 链路加密）、主从读写分离与多实例（InitNamed/Named），模型注册制自动迁移
- Redis 模块：字符串/列表/集合/有序集合/哈希/Lua/原子轮换，支持 TLS、ACL 用户名与多实例（InitNamed/Named）
- 日志模块：logrus + lumberjack 切割（`log` 模块）
- 可选能力：`requestid` 模块注入 `X-Request-Id`；`tenon.Bind` 请求绑定工具（仅返回错误）
- 统一响应封装与分页工具
- 配置封装（可选）：泛型配置文件加载（默认生成/字段补全/版本回写）、数据目录解析、CLI `--debug/--data/--config` 启动参数
- 内置 CLI：支持扩展自定义子命令（`AddCommand`/`Add`），`Serve`/`Run` 注册 server 子命令

## 快速开始

### 环境要求

- Go 1.26+

### 安装与运行

```bash
go get gopkg.761sama.com/tenon
```

### 一行装配（WebApp，推荐）

```go
package main

import (
	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon"
)

func main() {
	application := tenon.NewWebApp(tenon.WebAppOptions{
		HTTP: tenon.DefaultHTTPConfig(),
		// 主服务（HTTP）恒定启动，无需声明；Enable 只列可选模块，顺序即初始化顺序
		Enable: []string{tenon.ModuleLog},
	})
	s := application.Server()
	s.GET("/ping", func(c *gin.Context) {
		tenon.Success(c, gin.H{"msg": "pong"})
	})
	if err := application.Run(); err != nil {
		panic(err)
	}
}
```

启用数据库与 Redis（配置非法时初始化失败并返回错误，未列出的模块完全不参与）：

```go
application := tenon.NewWebApp(tenon.WebAppOptions{
	HTTP: cfg.HTTP, Log: cfg.Log, Database: cfg.DB, Redis: cfg.Redis,
	Enable: []string{
		tenon.ModuleLog,
		tenon.ModuleDatabase,
		tenon.ModuleRedis,
		tenon.ModuleRequestID,
	},
})
```

### 手工装配（自由度）

不使用 WebApp 时，各模块仍可逐个显式初始化（与 WebApp 内部走的是同一套 `Init`）：

```go
package main

import (
	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon"
	"gopkg.761sama.com/tenon/web"
)

func main() {
	if err := tenon.InitLog(tenon.LogConfig{Enable: false}, false); err != nil {
		panic(err)
	}
	server, err := web.New(tenon.DefaultHTTPConfig())
	if err != nil {
		panic(err)
	}
	server.GET("/ping", func(c *gin.Context) {
		tenon.Success(c, gin.H{"msg": "pong"})
	})
	if err := server.Run(); err != nil {
		panic(err)
	}
}
```

数据库与 Redis 的手工显式初始化：

```go
err := tenon.DB.Init(tenon.DatabaseConfig{Type: "sqlite3", Master: tenon.DBNodeConfig{DBFile: "data.db"}}, false)
tenon.DB.RegisterModels(new(User))
err = tenon.Redis.Init(tenon.RedisConfig{Host: "127.0.0.1", Port: 6379})
```

更多示例见 [example/](example/)：`basic`（手工装配）、`config`（配置驱动）、`cli`（CLI）、`app`（完整 WebApp 用法）。

## 项目结构

```
tenon.go / middleware.go / cli.go / webapp.go   框架门面（根包 tenon）
app/         装配层：模块注册表 + 启用清单 + 生命周期编排
conf/        配置结构定义（纯结构体）
bootstrap/   资源释放注册表
logx/        日志模块（显式初始化）
database/    数据库模块（gorm，显式初始化）
redis/       Redis 模块（显式初始化）
web/         Web 服务（gin 引擎封装）
common/      统一响应、错误码、分页
example/     使用示例
doc/         详细文档
```

## 文档

- 开发规范与 AI 工作流：见 [dev.md](dev.md)
- 详细文档：见 [doc/](doc/)（按主题拆分）
- 装配层与模块启用清单：见 [doc/app.md](doc/app.md)
- 框架设计与使用说明：见 [doc/framework.md](doc/framework.md)

## 许可证

无

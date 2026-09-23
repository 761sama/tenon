# tenon

> 通用 Go 服务框架：注册制设计，当前支持 Web 服务。

## 简介

tenon 是一个通用服务框架库，将 Web 服务、数据库、缓存、日志等基础设施以注册制方式封装，
使用方只需关注业务分层（router - controller - handler - model - dao），通过 `tenon.xxx` 快捷调用即可搭起服务骨架。

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
| 特殊约定 | 注册制设计（中间件、模型、初始化模块均可插拔注册）；允许引入成熟第三方依赖 |

## 功能特性

- Web 服务：基于 gin，支持 HTTP/HTTPS、CORS、优雅停机、路由组
- 中间件注册制：`tenon.RegMiddleware` 注册，`tenon.Middleware` 取用
- 数据库模块：gorm 封装，支持 sqlite3 / mysql 与主从读写分离，模型注册制自动迁移
- Redis 模块：go-redis 封装，`tenon.Redis` 显式初始化后直接调用（字符串/列表/集合/有序集合/哈希/Lua/原子轮换）
- 日志模块：logrus + lumberjack 切割
- 统一响应封装与分页工具
- 内置 CLI：默认提供 server 子命令，支持扩展自定义子命令

## 快速开始

### 环境要求

- Go 1.26+

### 安装与运行

```bash
go get gopkg.761sama.com/tenon
```

```go
package main

import (
	"github.com/gin-gonic/gin"
	"gopkg.761sama.com/tenon"
)

func main() {
	tenon.RegMiddleware("mark", func(c *gin.Context) { c.Next() })
	server := tenon.WebServer(tenon.DefaultConfig())
	server.Router("GET", "/ping", tenon.Middleware("mark"), func(c *gin.Context) {
		tenon.Success(c, gin.H{"msg": "pong"})
	})
	server.Run()
}
```

更多示例见 [example/](example/)。

## 项目结构

```
tenon.go / middleware.go / cli.go   框架门面（根包 tenon）
conf/        配置结构定义（纯结构体）
bootstrap/   注册制初始化模块
logx/        日志模块
database/    数据库模块（gorm）
redis/       Redis 模块
web/         Web 服务（gin 引擎封装）
common/      统一响应、错误码、分页
util/        工具函数
example/     使用示例
doc/         详细文档
```

## 文档

- 开发规范与 AI 工作流：见 [dev.md](dev.md)
- 详细文档：见 [doc/](doc/)（按主题拆分）

## 许可证

无

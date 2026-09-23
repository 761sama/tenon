// Package tenon 通用服务框架门面：提供 Web 服务创建、中间件注册、统一响应等快捷调用。
package tenon

import (
	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon/common"
	"gopkg.761sama.com/tenon/conf"
	"gopkg.761sama.com/tenon/redis"
	"gopkg.761sama.com/tenon/web"
)

// Version 框架版本。
const Version = "0.1.0"

// Redis 操作门面：需先通过 tenon.Redis.Init(tenon.RedisConfig{...}) 显式初始化。
var Redis = redis.Ops{}

// 配置类型别名，使用方通过 tenon.Config 等即可构造配置。
type (
	Config         = conf.Config         // 总配置
	HTTPConfig     = conf.HTTPConfig     // HTTP 服务配置
	LogConfig      = conf.LogConfig      // 日志配置
	DatabaseConfig = conf.DatabaseConfig // 数据库组配置
	DBNodeConfig   = conf.DBNodeConfig   // 数据库节点配置
	RedisConfig    = conf.RedisConfig    // Redis 配置
)

// 常用类型别名。
type (
	Context      = gin.Context      // 请求上下文
	HandlerFunc  = gin.HandlerFunc  // 控制器/中间件函数类型
	WebServerT   = web.WebServer    // Web 服务实例类型
	RouterGroup  = web.RouterGroup  // 路由组类型
	ErrorCode    = common.ErrorCode // 错误码
	PageRequest  = common.PageRequest  // 分页请求
)

// 构造默认配置，使用方在此基础上按需修改后传入 WebServer。
// 出参: 填充了合理默认值的 Config
func DefaultConfig() Config {
	return conf.Default()
}

// 创建 Web 服务实例：按配置初始化日志/数据库等模块并构建 gin 引擎。
// 入参: cfg (总配置)
// 出参: Web 服务实例
func WebServer(cfg Config) *WebServerT {
	return web.New(cfg)
}

// 成功响应快捷调用。
// 入参: c (请求上下文), results (业务数据，取第一个)
func Success(c *Context, results ...any) {
	common.SuccessResponse(c, results...)
}

// 错误响应快捷调用（错误码 -1）。
// 入参: c (请求上下文), message (提示信息)
func Error(c *Context, message string) {
	common.ErrorResponse(c, message)
}

// 使用预定义错误码响应。
// 入参: c (请求上下文), code (错误码)
func ErrorWithCode(c *Context, code ErrorCode) {
	common.ErrorResponseWithCode(c, code)
}

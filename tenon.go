// Package tenon 通用服务框架门面：提供 Web 服务创建、中间件注册、统一响应等快捷调用。
package tenon

import (
	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon/common"
	"gopkg.761sama.com/tenon/conf"
	"gopkg.761sama.com/tenon/database"
	"gopkg.761sama.com/tenon/logx"
	"gopkg.761sama.com/tenon/redis"
	"gopkg.761sama.com/tenon/web"
)

// Version 框架版本。
const Version = "0.1.0"

// 功能门面：数据库与 Redis 需显式初始化后方可使用。
var (
	// DB 数据库门面：先 DB.Init 显式初始化，再使用 RegisterModels/GetDB/Transaction 等
	DB = database.Ops{}
	// Redis 操作门面：先 Redis.Init 显式初始化，再使用 Get/Set 等操作
	Redis = redis.Ops{}
)

// 配置类型别名，使用方通过 tenon.HTTPConfig 等即可构造配置。
type (
	HTTPConfig     = conf.HTTPConfig     // HTTP 服务配置（WebServer 仅需此配置）
	LogConfig      = conf.LogConfig      // 日志配置
	DatabaseConfig = conf.DatabaseConfig // 数据库组配置
	DBNodeConfig   = conf.DBNodeConfig   // 数据库节点配置
	RedisConfig    = conf.RedisConfig    // Redis 配置
)

// 常用类型别名。
type (
	Context      = gin.Context        // 请求上下文
	HandlerFunc  = gin.HandlerFunc    // 控制器/中间件函数类型
	WebServerT   = web.WebServer      // Web 服务实例类型
	RouterGroup  = web.RouterGroup    // 路由组类型
	ErrorCode    = common.ErrorCode   // 错误码
	PageRequest  = common.PageRequest // 分页请求
	Tx           = database.Tx        // 事务连接类型
	DBInstance   = database.Instance  // 数据库实例（命名实例经 DB.Named 获取）
	RedisInstance = redis.Instance    // Redis 实例（命名实例经 Redis.Named 获取）
)

// 构造默认 HTTP 配置，使用方在此基础上按需修改后传入 WebServer。
// 出参: 填充了合理默认值的 HTTPConfig
func DefaultHTTPConfig() HTTPConfig {
	return conf.DefaultHTTPConfig()
}

// 通用配置文件加载（泛型，与具体配置结构解耦）：文件不存在时以默认值生成；
// 存在时先以默认值填充再反序列化覆盖（字段自动补全），随后回写文件（含版本回写）。
// 入参: path (配置文件路径), defaults (默认配置构造函数), version (配置版本，空字符串跳过)
// 出参: 配置对象, 来源 (conf.FromCreate/conf.FromLoad), 错误
func LoadConfig[T any](path string, defaults func() *T, version string) (*T, string, error) {
	return conf.Load(path, defaults, version)
}

// 将相对路径解析为基于数据目录的路径；绝对路径原样返回。
// 入参: dataDir (数据目录，空字符串表示工作目录), name (待解析的路径)
// 出参: 解析后的路径
func ResolveDataPath(dataDir, name string) string {
	return conf.ResolveDataPath(dataDir, name)
}

// 显式初始化日志模块：设置日志级别与文件切割输出；不调用时默认输出到标准错误。
// 入参: cfg (日志配置), debug (是否调试模式)
// 出参: 日志目录创建失败时返回错误
func InitLog(cfg LogConfig, debug bool) error {
	return logx.Init(cfg, debug)
}

// 创建 Web 服务实例：构建 gin 引擎（仅依赖 HTTP 配置）。
// 入参: cfg (HTTP 服务配置)
// 出参: Web 服务实例
func WebServer(cfg HTTPConfig) *WebServerT {
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

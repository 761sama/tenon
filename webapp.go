package tenon

import (
	"slices"
	"time"

	"gopkg.761sama.com/tenon/app"
	"gopkg.761sama.com/tenon/database"
	"gopkg.761sama.com/tenon/logx"
	"gopkg.761sama.com/tenon/redis"
	"gopkg.761sama.com/tenon/web"
)

// 可选模块名常量：供 WebAppOptions.Enable 引用（防拼写错误）；
// 自定义模块经 WebAppOptions.Custom 注册后用同一机制按名字符串启用。
const (
	ModuleLog       = "log"       // 日志模块：logx 文件切割输出（不启用则日志走 stderr）
	ModuleDatabase  = "database"  // 数据库模块：database.Init（含已注册模型的自动迁移）
	ModuleRedis     = "redis"     // Redis 模块：redis.Init
	ModuleRequestID = "requestid" // 请求 ID 模块：注入 X-Request-Id 中间件（无配置，默认不开）
)

// WebAppOptions Web 应用装配配置。
type WebAppOptions struct {
	HTTP     HTTPConfig     // HTTP 服务配置（主服务，恒定启动，不在 Enable 清单内）
	Log      LogConfig      // 日志模块配置（Enable 含 ModuleLog 时生效）
	Database DatabaseConfig // 数据库模块配置（Enable 含 ModuleDatabase 时生效）
	Redis    RedisConfig    // Redis 模块配置（Enable 含 ModuleRedis 时生效）
	Enable   []string       // 可选模块启用清单：顺序 = 初始化顺序；未列出 = 不加载；未知名/重复名 = 报错
	Custom   []app.Module   // 自定义模块：先注册进装配单元，再在 Enable 里按名字启用
}

// WebApp 预制 Web 应用：主服务（HTTP/HTTPS/QUIC）恒定启动，可选模块按 Enable 清单装配。
type WebApp struct {
	app    *app.App
	server *web.WebServer
	err    error // 构造期错误（HTTP 配置非法或 Enable 清单非法），Init/Run 时返回
}

// 创建预制 Web 应用：构建 Web 服务并预注册可选模块（log/database/redis/requestid），按 Enable 清单设置启用顺序；
// 构造期错误（如 TrustedProxies 非法、Enable 含未知名或重复名）在 Init/Run 时返回。
// requestid 中间件需在路由注册前挂载，故在构造时随 Enable 清单立即挂载，其模块 Init 为空操作。
// 入参: opt (装配配置)
// 出参: Web 应用实例
func NewWebApp(opt WebAppOptions) *WebApp {
	srv, err := web.New(opt.HTTP)
	a := app.New("web")
	w := &WebApp{app: a, server: srv, err: err}
	if srv != nil && slices.Contains(opt.Enable, ModuleRequestID) {
		srv.Engine().Use(web.RequestIDMiddleware())
	}
	a.Add(app.Module{Name: ModuleLog, Init: func() error {
		return logx.Init(opt.Log, opt.HTTP.Debug)
	}})
	a.Add(app.Module{Name: ModuleDatabase, Init: func() error {
		return database.Init(opt.Database, opt.HTTP.Debug)
	}, Release: database.CloseAll})
	a.Add(app.Module{Name: ModuleRedis, Init: func() error {
		return redis.Init(opt.Redis)
	}, Release: redis.CloseAll})
	a.Add(app.Module{Name: ModuleRequestID, Init: func() error { return nil }})
	for _, mod := range opt.Custom {
		a.Add(mod)
	}
	if e := a.Enable(opt.Enable...); w.err == nil {
		w.err = e
	}
	if srv != nil {
		a.OnRun(func() error { return srv.Start() })
		a.OnStop(func(timeout time.Duration) { srv.Stop(timeout) })
	}
	a.SetStopTimeout(opt.HTTP.ShutdownTimeout)
	return w
}

// 取底层 Web 服务，用于注册路由与中间件（Use 必须在注册路由之前调用）。
// 出参: Web 服务实例（构造失败时为 nil）
func (a *WebApp) Server() *web.WebServer {
	return a.server
}

// 初始化：按 Enable 清单顺序初始化各模块（任一失败则逆序释放已成功初始化的模块）。
// 出参: 构造期错误或首个失败模块的错误
func (a *WebApp) Init() error {
	if a.err != nil {
		return a.err
	}
	return a.app.Init()
}

// 阻塞运行：初始化 -> 启动主服务 -> 等待 SIGINT/SIGTERM -> 停主服务 -> 逆序释放模块。
// 出参: 构造期错误、初始化错误或主服务启动错误
func (a *WebApp) Run() error {
	if a.err != nil {
		return a.err
	}
	return a.app.Run()
}

// 停止应用：停主服务并逆序释放模块。幂等：重复调用安全。
// 入参: timeout (优雅停机超时时间)
func (a *WebApp) Stop(timeout time.Duration) {
	a.app.Stop(timeout)
}

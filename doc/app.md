# tenon 装配层（app）说明

## 定位

`app` 是**装配单元**：持有「模块注册表 + 启用清单」与生命周期编排，不只服务 Web。
预制 app（当前交付 `WebApp`）= **主服务（固定）+ 可选模块集**。

- 主服务（如 `WebApp` 的 HTTP/HTTPS/QUIC）**恒定启动**，不设开关、不出现在 `Enable` 里
- 可选模块以注册制选择性加载：`Add` 只注册（不执行），`Enable` 决定启用哪些、按什么顺序初始化
- `web/` 包保持纯 HTTP，不感知 app 层与模块编排

## 核心概念

```go
// Module 模块定义：名字 + 初始化 + 释放（Release 可为 nil）；Init 是闭包，配置在构造模块时捕获
type Module struct {
	Name    string
	Init    func() error
	Release func()
}
```

`app.App` 装配单元 API：

```go
a := app.New("web")
a.Add(app.Module{Name: "m", Init: ..., Release: ...}) // 注册（不执行）；同名覆盖实现，保留原注册顺序位置
a.Has("m")                                            // 是否已注册
a.Names()                                             // 已注册模块名（注册顺序）
err := a.Enable("m1", "m2")                           // 设置启用清单：顺序 = 初始化顺序
a.Enabled()                                           // 当前启用清单
a.OnRun(func() error { ... })                         // 注册运行体（非阻塞启动，如 web.Start）
a.OnStop(func(timeout time.Duration) { ... })         // 注册停机体（按注册逆序执行）
err = a.Init()                                        // 按启用清单顺序初始化；成功后幂等
err = a.Run()                                         // Init -> OnRun -> 等信号 -> OnStop(逆序) -> 模块逆序释放
a.Stop(5 * time.Second)                               // 停止（幂等：重复调用安全）
a.IsRunning()                                         // 运行状态
```

## 启用清单（Enable）语义

| 场景 | 行为 |
|------|------|
| `Enable` 不传 / 为空 | 所有模块默认禁用：不加载任何模块、不读任何模块配置、不报错；**主服务照常启动** |
| 加载顺序 | 严格按 `Enable` 列表顺序（`{"redis","database","log"}` 就先初始化 Redis） |
| 出现未注册的模块名 | 显式报错：`unknown module "x"; known modules: ...`（列出全部已知模块名），绝不静默跳过 |
| 出现重复的模块名 | 报错（配置错误尽早暴露） |
| 显式列出 = 明确要开 | 该模块配置非法 → 初始化失败并返回错误（不静默降级） |
| 未列出的模块 | 完全不参与（配置段可留空、可留着不管） |

生命周期编排：

- `Init()` 按启用清单顺序执行各模块 `Init`；**任一失败 → 逆序释放已成功初始化的模块 → 返回错误**（不留半初始化状态）
- `Release` 为 nil 的模块跳过；`Stop()` 幂等
- `Run()` = `Init()` → 依次执行运行体 → 等 `SIGINT`/`SIGTERM` → 停机体逆序执行 → 模块按初始化逆序释放
- `Init` 失败时 `Run()` 直接返回错误，不进入等待信号；运行体启动失败时已初始化的模块按逆序释放
- `App.Run` 等信号后的停机超时默认 5s，可经 `SetStopTimeout` 调整（`WebApp` 自动采用 `HTTPConfig.ShutdownTimeout`）

## WebApp 预制装配

```go
application := tenon.NewWebApp(tenon.WebAppOptions{
	HTTP: cfg.HTTP, Log: cfg.Log, Database: cfg.DB, Redis: cfg.Redis,
	Enable: []string{
		tenon.ModuleLog,
		tenon.ModuleDatabase,
		tenon.ModuleRedis,
		tenon.ModuleRequestID,
	},
	Custom: []app.Module{{Name: "mymod", Init: ..., Release: ...}}, // 自定义模块：先注册，再在 Enable 里按名启用
})
s := application.Server()             // *web.WebServer，gin 类型直接用
s.Use(authMiddleware)                 // ⚠️ Use 必须在注册路由之前调用
s.GET("/api/library", listHandler)
v1 := s.Group("/v1")
v1.GET("/ping", pingHandler)
err := application.Run()              // 初始化 -> 监听 -> 等信号 -> 停机 -> 释放
```

预注册的可选模块表（主服务不在其列）：

| 模块名 | 常量 | 作用 | 配置来源 |
|--------|------|------|---------|
| `log` | `tenon.ModuleLog` | `logx.Init`（不启用则日志走 stderr） | `WebAppOptions.Log` |
| `database` | `tenon.ModuleDatabase` | `database.Init`（含已注册模型的自动迁移） | `WebAppOptions.Database` |
| `redis` | `tenon.ModuleRedis` | `redis.Init` | `WebAppOptions.Redis` |
| `requestid` | `tenon.ModuleRequestID` | 注入 `X-Request-Id`（默认不开） | 无 |

- 编排顺序固定：**先按 `Enable` 清单初始化各模块 → 再启动主服务**（保证 DB / Redis 先就绪）；停机时逆序：主服务 → 模块
- `Enable` 为空时 `WebApp` 仍是正常运行的 Web 服务（只是没有 DB / Redis / 文件日志 / requestid）
- `requestid` 中间件需在路由注册前挂载，故在 `NewWebApp` 构造时随 `Enable` 清单立即挂载（其模块 `Init` 为空操作）
- 自定义模块与预制模块同名时按覆盖规则处理（覆盖实现，保留原注册顺序位置）

## 扩展新可选能力

新增模块 = 在预制 app 注册 + 在根包加 `ModuleXxx` 常量，`Enable` 机制不用改：

```go
// 1. 根包加常量（防拼写；自定义模块也可直接用字符串字面量）
const ModuleCache = "cache"
// 2. 预制 app 构造时注册（闭包捕获配置）
a.Add(app.Module{Name: ModuleCache, Init: func() error { return cache.Init(opt.Cache) }, Release: cache.Close})
// 3. 使用方在 Enable 里按名启用
```

## 与 bootstrap 的关系

- `bootstrap` 只保留资源释放注册表：`RegisterRelease(name, fn)` / `Release()`（停机时按注册逆序执行）
- `database` / `redis` 的显式 `Init` 成功后仍自动注册释放到 `bootstrap`（保持现有行为），`web.Server.Stop` 时统一执行
- `WebApp` 场景下模块释放由装配层编排（模块 `Release` = `CloseAll`，与 `bootstrap` 释放幂等兼容）
- 旧的 `bootstrap.RegisterInitModule` / `Init` / `TinyInit` 已移除，模块初始化注册由 `app/` 接管

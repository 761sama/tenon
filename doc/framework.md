# tenon 框架设计与使用说明

## 架构

```
使用方业务分层：router -> controller -> handler -> model -> dao
框架支撑：web(gin) 仅依赖 HTTP 配置；logx / database / redis 按需显式初始化
```

核心设计原则：**注册制 + 按需启用 + 显式初始化**。

- Web 服务：`tenon.WebServer(tenon.HTTPConfig{...})` 只接收 HTTP/Web 相关配置
- 数据库：`tenon.DB.Init(...)` 显式初始化后使用，未初始化时不影响 Web 服务运行
- Redis：`tenon.Redis.Init(...)` 显式初始化后使用
- 日志：`tenon.InitLog(...)` 显式初始化文件切割输出，不调用时输出到标准错误
- 模型注册：`tenon.DB.RegisterModels(...)`，数据库初始化时自动迁移；已初始化后注册则立即迁移
- 中间件注册：`tenon.RegMiddleware("名称", mw)`，取用 `tenon.Middleware("名称")`
- 初始化模块扩展点：`bootstrap.RegisterInitModule("名称", fn)` 注册，`bootstrap.Init()` 按需执行
- 资源释放注册：`bootstrap.RegisterRelease("名称", fn)`，停机时按注册逆序执行（数据库/Redis 初始化时自动注册）

## 快速开始

### 直接运行

```go
cfg := tenon.DefaultHTTPConfig()     // 默认 HTTP 配置，纯结构体，可按需修改
cfg.Port = 8080
server := tenon.WebServer(cfg)       // 创建服务（仅构建 gin 引擎）
server.Router("GET", "/ping", tenon.Middleware("mark"), controller)
v1 := server.Group("/v1")            // 路由组
v1.Router("POST", "/users", createUserCtrl)
server.Run()                         // 阻塞运行，收到 SIGINT/SIGTERM 后优雅停机
```

### 内置 CLI

```go
cli := tenon.NewCli("myapp", "我的服务")
cli.AddCommand("version", "打印版本", func() { println("v1.0.0") })
cli.Run(server)                      // 注册 server 子命令并执行
```

## HTTP 配置说明（tenon.HTTPConfig）

| 字段 | 说明 |
|------|------|
| Debug | 调试模式：gin 调试模式、错误不脱敏 |
| Port / HTTPSPort | 监听端口，-1 表示禁用对应协议 |
| CertFile / KeyFile | HTTPS 证书（相对路径基于工作目录解析） |
| TrustedProxies | 可信代理，防止 X-Forwarded-For 伪造 |
| AllowOrigins/Methods/Headers | CORS 配置 |
| ReadTimeout / WriteTimeout | 读写超时，0 表示使用默认值 |

## 数据库

```go
err := tenon.DB.Init(tenon.DatabaseConfig{
	Type:   "mysql", // 或 "sqlite3"（Master.DBFile 指定文件路径）
	Master: tenon.DBNodeConfig{Host: "127.0.0.1", Port: 3306, User: "u", Password: "p", Name: "db"},
	// Replicas: []tenon.DBNodeConfig{...}, // 从库，经 dbresolver 随机读写分离
}, false)
tenon.DB.RegisterModels(new(User), new(Order))   // 初始化前后注册均可
db := tenon.DB.GetDB()
tenon.DB.Transaction(func(tx tenon.Tx) error { ... })
tenon.DB.GetDBWithTx(tx)                          // dao 层兼容事务
tenon.DB.ColumnName("name")                       // 列名引用兼容
tenon.DB.IsAvailable()                            // 是否已初始化
```

## Redis

Redis 采用**显式初始化**，不随 `tenon.WebServer` 自动初始化：

```go
err := tenon.Redis.Init(tenon.RedisConfig{Host: "127.0.0.1", Port: 6379})
tenon.Redis.Set("value", "key1", "key2")        // 键片段以冒号拼接为 key1:key2
val, _ := tenon.Redis.Get("key1", "key2")
tenon.Redis.SetWithTTL("v", time.Minute, "k")   // 带过期时间
tenon.Redis.LRPush([]string{"a"}, "list")       // 列表
tenon.Redis.SAdd([]string{"x"}, "set")          // 集合
tenon.Redis.ZAdd(zs, "zset")                    // 有序集合
tenon.Redis.HSet(map[string]any{"f": "v"}, "h") // 哈希
tenon.Redis.Incr("counter")                     // 自增
tenon.Redis.Eval(script, args, "key")           // Lua 脚本
tenon.Redis.Rotate(data, ttl, oldKey, newKey)   // 原子轮换（并发安全的旧键消费）
tenon.Redis.Client()                            // 获取底层 go-redis 客户端
```

- 未初始化或连接不可用时，各操作返回哨兵错误 `redis.ErrRedisUnavailable`，供调用方降级处理
- 初始化成功后自动注册停机释放，`server.Stop` 时会一并关闭连接

## 日志

```go
tenon.InitLog(tenon.LogConfig{
	Enable: true, Name: "log/app.log",
	MaxSize: 50, MaxBackups: 30, MaxAge: 28,
}, cfg.Debug)
```

## 中间件

```go
tenon.RegMiddleware("auth", authMiddleware)          // 注册
server.Router("GET", "/me", tenon.Middleware("auth"), meCtrl)  // 按名取用
server.Router("GET", "/ping", directMiddleware, pingCtrl)      // 也可直接传函数
```

- Router 签名：`Router(method, path, 中间件..., 控制器)`，最后一个参数为控制器
- 未注册的名称调用 `tenon.Middleware` 会 panic（启动期错误尽早暴露）
- method 支持 GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS/ANY

## 统一响应

```go
tenon.Success(c, data)                 // {"code":0,"message":"成功","result":data}
tenon.Error(c, "提示")                  // {"code":-1,...}
tenon.ErrorWithCode(c, code)           // 预定义错误码，见 common/code.go
common.FullResponse(...)               // 完整控制
common.MaskError(err, "操作失败")       // 非调试模式错误脱敏
common.PageSizeCheck(page, pageSize)   // 分页参数修正
```

## 优雅停机

`server.Run()` 阻塞等待 SIGINT/SIGTERM，收到信号后：

1. 按超时优雅关闭 HTTP/HTTPS 监听
2. 按注册逆序执行 `bootstrap` 释放函数（Redis -> 数据库）

也可手动控制：`server.Start()` 非阻塞启动，`server.Stop(timeout)` 停止。

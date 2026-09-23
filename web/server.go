package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quic-go/quic-go/http3"
	log "github.com/sirupsen/logrus"

	"gopkg.761sama.com/tenon/bootstrap"
	"gopkg.761sama.com/tenon/common"
	"gopkg.761sama.com/tenon/conf"
)

// WebServer Web 服务实例。
type WebServer struct {
	cfg       conf.HTTPConfig
	engine    *gin.Engine
	running   atomic.Bool
	httpSrv   *http.Server
	httpsSrv  *http.Server
	quicSrv   *http3.Server
	stopOnce  sync.Once
}

// 创建 Web 服务实例：构建 gin 引擎。
// 日志/数据库/Redis 不随服务自动初始化，需分别通过
// tenon.InitLog / tenon.DB.Init / tenon.Redis.Init 显式初始化。
// 入参: cfg (HTTP 服务配置)
// 出参: Web 服务实例
func New(cfg conf.HTTPConfig) *WebServer {
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	common.SetDebug(cfg.Debug)
	engine := gin.New()
	engine.ContextWithFallback = true
	if err := engine.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Errorf("failed to set trusted proxies: %s", err.Error())
	}
	engine.Use(gin.LoggerWithWriter(log.StandardLogger().Out))
	engine.Use(RecoveryMiddleware())
	engine.Use(CorsMiddleware(cfg))
	engine.Use(MaxBodySizeMiddleware(cfg.MaxBodySize))
	engine.NoRoute(NoRouteHandle)
	// QUIC 启用时注册 Alt-Svc 中间件（须在任何 ListenAndServe 之前完成）
	if cfg.EnableQUIC {
		engine.Use(AltSvcMiddleware(cfg.HTTPSPort))
	}
	return &WebServer{cfg: cfg, engine: engine}
}

// 获取底层 gin 引擎，用于注册框架未封装的能力。
// 出参: gin 引擎
func (s *WebServer) Engine() *gin.Engine {
	return s.engine
}

// 注册路由：前若干参数为中间件函数，最后一个参数为控制器。
// 入参: method (HTTP 方法，如 GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS/ANY),
//
//	path (路由路径), handlers (中间件函数与控制器的有序列表)
func (s *WebServer) Router(method, path string, handlers ...gin.HandlerFunc) {
	handle(s.engine, method, path, handlers)
}

// 创建路由组，组内路由共享前缀与中间件。
// 入参: prefix (路由前缀), handlers (组级中间件函数列表)
// 出参: 路由组
func (s *WebServer) Group(prefix string, handlers ...gin.HandlerFunc) *RouterGroup {
	return &RouterGroup{group: s.engine.Group(prefix, handlers...)}
}

// 非阻塞启动服务。
// 出参: 启动错误（端口均禁用或 HTTPS 配置不完整时返回错误）
func (s *WebServer) Start() error {
	started := false
	if s.cfg.Port != -1 {
		s.httpSrv = s.buildServer(s.cfg.Address, s.cfg.Port, false)
		started = true
	}
	if s.cfg.HTTPSPort != -1 {
		if s.cfg.CertFile == "" || s.cfg.KeyFile == "" {
			return fmt.Errorf("https enabled but cert_file or key_file is empty")
		}
		s.httpsSrv = s.buildServer(s.cfg.Address, s.cfg.HTTPSPort, true)
		if s.cfg.EnableQUIC {
			base := fmt.Sprintf("%s:%d", s.cfg.Address, s.cfg.HTTPSPort)
			s.quicSrv = s.startQUIC(base, s.cfg.CertFile, s.cfg.KeyFile)
		}
		started = true
	}
	if !started {
		return fmt.Errorf("no http listener enabled")
	}
	s.running.Store(true)
	return nil
}

// 阻塞运行服务：启动后等待 SIGINT/SIGTERM 信号，收到后优雅停机并释放资源。
// 出参: 启动错误
func (s *WebServer) Run() error {
	if err := s.Start(); err != nil {
		return err
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	s.Stop(5 * time.Second)
	return nil
}

// 停止服务：优雅关闭 HTTP/HTTPS 监听并释放数据库、Redis 等资源。
// 入参: timeout (优雅停机超时时间)
func (s *WebServer) Stop(timeout time.Duration) {
	s.stopOnce.Do(func() {
		log.Infof("正在关闭服务器...")
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		var wg sync.WaitGroup
		shutdown := func(srv *http.Server) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := srv.Shutdown(ctx); err != nil {
					log.Errorf("http server shutdown err: %s", err)
				}
			}()
		}
		if s.httpSrv != nil {
			shutdown(s.httpSrv)
		}
		if s.httpsSrv != nil {
			shutdown(s.httpsSrv)
		}
		s.stopQUIC(ctx)
		wg.Wait()
		bootstrap.Release()
		s.running.Store(false)
		log.Infof("服务器已关闭")
	})
}

// 服务是否正在运行。
// 出参: 运行状态
func (s *WebServer) IsRunning() bool {
	return s.running.Load()
}

// 构建 HTTP 服务器（纯构造，不启动监听）：应用超时配置，零值回落到安全默认值。
// 入参: address (监听地址), port (端口)
// 出参: http.Server 实例
func (s *WebServer) newHTTPServer(address string, port int) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", address, port),
		Handler:           s.engine,
		ReadHeaderTimeout: durationOrDefault(s.cfg.ReadHeaderTimeout, 10*time.Second),
		ReadTimeout:       durationOrDefault(s.cfg.ReadTimeout, 30*time.Second),
		WriteTimeout:      durationOrDefault(s.cfg.WriteTimeout, 30*time.Second),
		IdleTimeout:       durationOrDefault(s.cfg.IdleTimeout, 120*time.Second),
	}
}

// 时长零值回落到默认值（零值在 http.Server 中表示无超时，存在慢连接风险）。
// 入参: v (配置值), def (默认值)
// 出参: 实际使用的时长
func durationOrDefault(v, def time.Duration) time.Duration {
	if v <= 0 {
		return def
	}
	return v
}

// 构建并启动单个 HTTP/HTTPS 监听。
// 入参: address (监听地址), port (端口), tls (是否启用 TLS)
// 出参: http.Server 实例
func (s *WebServer) buildServer(address string, port int, tls bool) *http.Server {
	srv := s.newHTTPServer(address, port)
	base := srv.Addr
	go func() {
		var err error
		if tls {
			log.Infof("start HTTPS server @ %s", base)
			err = srv.ListenAndServeTLS(s.cfg.CertFile, s.cfg.KeyFile)
		} else {
			log.Infof("start HTTP server @ %s", base)
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Errorf("failed to start server @ %s: %s", base, err.Error())
		}
	}()
	return srv
}

// 按方法名分发路由注册，未知方法直接 panic（启动期编程错误应尽早暴露）。
// 入参: rg (gin 路由接口), method (HTTP 方法), path (路由路径), handlers (处理器列表)
func handle(rg gin.IRouter, method, path string, handlers []gin.HandlerFunc) {
	if len(handlers) == 0 {
		panic(fmt.Sprintf("tenon: route %s %s requires at least one handler", method, path))
	}
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions:
		rg.Handle(method, path, handlers...)
	case "ANY":
		rg.Any(path, handlers...)
	default:
		panic(fmt.Sprintf("tenon: unsupported http method: %s", method))
	}
}

package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quic-go/quic-go/http3"
	log "github.com/sirupsen/logrus"
)

// Alt-Svc 中间件：对 TLS 请求附加 Alt-Svc 响应头，引导客户端升级到 HTTP/3。
// 入参: port (HTTPS/QUIC 端口)
// 出参: gin 中间件函数
func AltSvcMiddleware(port int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.TLS != nil {
			c.Header("Alt-Svc", fmt.Sprintf(`h3=":%d"; ma=86400`, port))
		}
		c.Next()
	}
}

// 启动 QUIC (HTTP/3) 服务器，与 HTTPS 同地址监听。
// 入参: base (监听地址), certFile (TLS 证书路径), keyFile (TLS 私钥路径)
// 出参: http3.Server 实例
func (s *WebServer) startQUIC(base, certFile, keyFile string) *http3.Server {
	quicSrv := &http3.Server{Addr: base, Handler: s.engine}
	go func() {
		log.Infof("start HTTP3 (quic) server @ %s", base)
		err := quicSrv.ListenAndServeTLS(certFile, keyFile)
		if err != nil && err != http.ErrServerClosed {
			log.Errorf("failed to start http3 (quic): %s", err.Error())
		}
	}()
	return quicSrv
}

// 停止 QUIC 服务器。
// 入参: ctx (停机超时上下文)
func (s *WebServer) stopQUIC(ctx context.Context) {
	if s.quicSrv == nil {
		return
	}
	if err := s.quicSrv.Shutdown(ctx); err != nil {
		log.Errorf("HTTP3 (quic) server shutdown err: %s", err)
	}
	s.quicSrv = nil
}

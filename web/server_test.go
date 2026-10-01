package web

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/quic-go/quic-go/http3"

	"gopkg.761sama.com/tenon/conf"
)

// 生成 127.0.0.1 自签证书与私钥文件。
// 出参: 证书文件路径、私钥文件路径
func genSelfSignedCert(t *testing.T) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %s", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "tenon-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %s", err)
	}
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")
	certOut, err := os.Create(certFile)
	if err != nil {
		t.Fatalf("failed to write cert: %s", err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("failed to encode cert: %s", err)
	}
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal key: %s", err)
	}
	keyOut, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("failed to write key: %s", err)
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		t.Fatalf("failed to encode key: %s", err)
	}
	return certFile, keyFile
}

// 获取空闲端口。
// 出参: 可用端口号
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %s", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// 验证 HTTPS + QUIC：TLS 请求携带 Alt-Svc 头，HTTP/3 客户端可经 QUIC 完成请求。
func TestHTTPSAndQUIC(t *testing.T) {
	certFile, keyFile := genSelfSignedCert(t)
	port := freePort(t)
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	cfg.HTTPSPort = port
	cfg.CertFile = certFile
	cfg.KeyFile = keyFile
	cfg.EnableQUIC = true
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	srv.Router("GET", "/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"msg": "pong"})
	})
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %s", err)
	}
	defer srv.Stop(3 * time.Second)
	time.Sleep(500 * time.Millisecond)
	base := fmt.Sprintf("https://127.0.0.1:%d", port)
	// HTTPS 请求：校验 Alt-Svc 头
	httpsClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	resp, err := httpsClient.Get(base + "/ping")
	if err != nil {
		t.Fatalf("https request failed: %s", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.Contains(string(body), "pong") {
		t.Fatalf("unexpected https response: %d %s", resp.StatusCode, body)
	}
	altSvc := resp.Header.Get("Alt-Svc")
	if !strings.Contains(altSvc, fmt.Sprintf(`h3=":%d"`, port)) {
		t.Fatalf("missing Alt-Svc header: %q", altSvc)
	}
	// TLS 请求应附加 HSTS 安全头
	if resp.Header.Get("Strict-Transport-Security") == "" {
		t.Fatal("https response should carry HSTS header")
	}
	// QUIC 请求：http3 客户端直连
	rt := &http3.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	defer rt.Close()
	req, err := http.NewRequest("GET", base+"/ping", nil)
	if err != nil {
		t.Fatalf("failed to build request: %s", err)
	}
	qresp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("quic request failed: %s", err)
	}
	qbody, _ := io.ReadAll(qresp.Body)
	qresp.Body.Close()
	if qresp.StatusCode != 200 || !strings.Contains(string(qbody), "pong") {
		t.Fatalf("unexpected quic response: %d %s", qresp.StatusCode, qbody)
	}
	if qresp.Proto != "HTTP/3.0" {
		t.Fatalf("unexpected proto: %s", qresp.Proto)
	}
}

// 验证启用 QUIC 但未启用 HTTPS 时不启动 QUIC。
func TestQUICRequiresHTTPS(t *testing.T) {
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	cfg.HTTPSPort = -1
	cfg.EnableQUIC = true
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	if err := srv.Start(); err == nil {
		t.Fatal("start should fail when no listener enabled")
	}
	if srv.quicSrv != nil {
		t.Fatal("quic should not start without https")
	}
}

// 验证请求体大小限制：超限返回 413，未超限正常处理。
func TestMaxBodySize(t *testing.T) {
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	cfg.MaxBodySize = 16
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	srv.Router("POST", "/echo", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.String(200, string(body))
	})
	// 超限（Content-Length 已知）
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(strings.Repeat("x", 32)))
	srv.Engine().ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body should be 413: %d", w.Code)
	}
	// 未超限
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader("ok"))
	srv.Engine().ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Fatalf("normal body should pass: %d %s", w.Code, w.Body.String())
	}
	// 超限（无 Content-Length，chunked 由 MaxBytesReader 兜底）
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/echo", io.NopCloser(strings.NewReader(strings.Repeat("x", 32))))
	req.ContentLength = -1
	srv.Engine().ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("chunked oversized body should be 413: %d", w.Code)
	}
}

// 验证 CORS：默认关闭（无 CORS 头），显式配置来源时启用，["*"] 允许所有来源。
func TestCors(t *testing.T) {
	newServer := func(origins ...string) *WebServer {
		cfg := conf.DefaultHTTPConfig()
		cfg.Port = -1
		cfg.AllowOrigins = origins
		srv, err := New(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %s", err)
		}
		srv.Router("GET", "/ping", func(c *gin.Context) { c.String(200, "pong") })
		return srv
	}
	preflight := func(srv *WebServer, origin string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "GET")
		srv.Engine().ServeHTTP(w, req)
		return w
	}
	// 默认关闭：不附加 CORS 头，OPTIONS 由 NoRoute 兜底
	srv := newServer()
	w := preflight(srv, "https://evil.example")
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("cors should be disabled by default: %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	// 指定来源：仅该来源放行
	srv = newServer("https://a.example")
	w = preflight(srv, "https://a.example")
	if w.Header().Get("Access-Control-Allow-Origin") != "https://a.example" {
		t.Fatalf("allowed origin should pass: %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	w = preflight(srv, "https://evil.example")
	if w.Header().Get("Access-Control-Allow-Origin") == "https://evil.example" {
		t.Fatal("disallowed origin should not pass")
	}
	// ["*"] 允许所有来源
	srv = newServer("*")
	w = preflight(srv, "https://any.example")
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("wildcard origin should return *: %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

// 验证安全响应头：默认附加 nosniff/禁嵌入/no-referrer（纯 HTTP 不带 HSTS），可配置关闭。
func TestSecureHeaders(t *testing.T) {
	newServer := func(disable bool) *WebServer {
		cfg := conf.DefaultHTTPConfig()
		cfg.Port = -1
		cfg.DisableSecureHeaders = disable
		srv, err := New(cfg)
		if err != nil {
			t.Fatalf("failed to create server: %s", err)
		}
		srv.Router("GET", "/ping", func(c *gin.Context) { c.String(200, "pong") })
		return srv
	}
	// 默认开启：三个基础头存在，纯 HTTP 不附加 HSTS
	srv := newServer(false)
	w := httptest.NewRecorder()
	srv.Engine().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	h := w.Header()
	if h.Get("X-Content-Type-Options") != "nosniff" || h.Get("X-Frame-Options") != "DENY" || h.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("missing secure headers: %v", h)
	}
	if h.Get("Strict-Transport-Security") != "" {
		t.Fatal("plain http should not carry HSTS")
	}
	// 配置关闭：不附加任何安全头
	srv = newServer(true)
	w = httptest.NewRecorder()
	srv.Engine().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ping", nil))
	h = w.Header()
	if h.Get("X-Content-Type-Options") != "" || h.Get("X-Frame-Options") != "" || h.Get("Referrer-Policy") != "" {
		t.Fatalf("secure headers should be disabled: %v", h)
	}
}

// 验证 TLS 最低版本配置：默认 TLS 1.2，非法值构造期报错，"1.3" 拒绝 TLS 1.2 握手。
func TestTLSMinVersion(t *testing.T) {
	// 解析行为：默认/1.2 -> TLS1.2，1.3 -> TLS1.3，非法报错
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	srv, err := New(cfg)
	if err != nil || srv.tlsMin != tls.VersionTLS12 {
		t.Fatalf("default should be TLS 1.2: %d, %v", srv.tlsMin, err)
	}
	cfg.TLSMinVersion = "1.3"
	if srv, err = New(cfg); err != nil || srv.tlsMin != tls.VersionTLS13 {
		t.Fatalf("explicit 1.3 should be respected: %d, %v", srv.tlsMin, err)
	}
	cfg.TLSMinVersion = "1.1"
	if _, err = New(cfg); err == nil {
		t.Fatal("unsupported tls_min_version should return error")
	}
	// 真实握手：TLSMinVersion=1.3 的服务拒绝 TLS 1.2 客户端，接受 TLS 1.3 客户端
	certFile, keyFile := genSelfSignedCert(t)
	port := freePort(t)
	cfg = conf.DefaultHTTPConfig()
	cfg.Port = -1
	cfg.HTTPSPort = port
	cfg.CertFile = certFile
	cfg.KeyFile = keyFile
	cfg.TLSMinVersion = "1.3"
	srv, err = New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	srv.Router("GET", "/ping", func(c *gin.Context) { c.String(200, "pong") })
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start server: %s", err)
	}
	defer srv.Stop(3 * time.Second)
	time.Sleep(500 * time.Millisecond)
	base := fmt.Sprintf("https://127.0.0.1:%d", port)
	client := func(max uint16) *http.Client {
		return &http.Client{
			Timeout:   5 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true, MaxVersion: max}},
		}
	}
	if _, err := client(tls.VersionTLS12).Get(base + "/ping"); err == nil {
		t.Fatal("TLS 1.2 client should be rejected when min version is 1.3")
	}
	resp, err := client(tls.VersionTLS13).Get(base + "/ping")
	if err != nil {
		t.Fatalf("TLS 1.3 client should succeed: %s", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || resp.TLS == nil || resp.TLS.Version != tls.VersionTLS13 {
		t.Fatalf("unexpected TLS 1.3 response: %d %+v", resp.StatusCode, resp.TLS)
	}
}

// 验证超时配置：零值回落默认值，显式配置生效。
func TestTimeoutConfig(t *testing.T) {
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	httpSrv := srv.newHTTPServer("127.0.0.1", 8080)
	if httpSrv.ReadHeaderTimeout != 10*time.Second || httpSrv.IdleTimeout != 120*time.Second {
		t.Fatalf("unexpected default timeouts: header=%v idle=%v", httpSrv.ReadHeaderTimeout, httpSrv.IdleTimeout)
	}
	cfg.ReadHeaderTimeout = 3 * time.Second
	cfg.IdleTimeout = 60 * time.Second
	if srv, err = New(cfg); err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	httpSrv = srv.newHTTPServer("127.0.0.1", 8080)
	if httpSrv.ReadHeaderTimeout != 3*time.Second || httpSrv.IdleTimeout != 60*time.Second {
		t.Fatalf("explicit timeouts should be respected: header=%v idle=%v", httpSrv.ReadHeaderTimeout, httpSrv.IdleTimeout)
	}
	// 零值配置也应回落默认值（而非无超时）
	if srv, err = New(conf.HTTPConfig{Port: -1}); err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	httpSrv = srv.newHTTPServer("127.0.0.1", 8080)
	if httpSrv.ReadHeaderTimeout <= 0 || httpSrv.ReadTimeout <= 0 || httpSrv.WriteTimeout <= 0 || httpSrv.IdleTimeout <= 0 {
		t.Fatal("zero config should fall back to safe defaults")
	}
	// 停机超时：默认 5s，显式配置生效，零值回落
	if conf.DefaultHTTPConfig().ShutdownTimeout != 5*time.Second {
		t.Fatalf("unexpected default shutdown timeout: %v", conf.DefaultHTTPConfig().ShutdownTimeout)
	}
	if got := durationOrDefault(0, 5*time.Second); got != 5*time.Second {
		t.Fatalf("zero shutdown timeout should fall back: %v", got)
	}
	if got := durationOrDefault(10*time.Second, 5*time.Second); got != 10*time.Second {
		t.Fatalf("explicit shutdown timeout should be respected: %v", got)
	}
}

// 验证非法 TrustedProxies 配置在 New 时显式返回错误（不再静默降级）。
func TestNewInvalidTrustedProxies(t *testing.T) {
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	cfg.TrustedProxies = []string{"not-a-cidr"}
	if _, err := New(cfg); err == nil {
		t.Fatal("invalid trusted proxies should return error")
	}
}

// 验证 8 个 HTTP 方法糖与 ANY 等价于 Router 动态方法。
func TestMethodSugar(t *testing.T) {
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	ok := func(c *gin.Context) { c.String(200, "ok") }
	srv.GET("/get", ok)
	srv.POST("/post", ok)
	srv.PUT("/put", ok)
	srv.DELETE("/delete", ok)
	srv.PATCH("/patch", ok)
	srv.HEAD("/head", ok)
	srv.OPTIONS("/options", ok)
	srv.ANY("/any", ok)
	cases := []struct{ method, path string }{
		{http.MethodGet, "/get"},
		{http.MethodPost, "/post"},
		{http.MethodPut, "/put"},
		{http.MethodDelete, "/delete"},
		{http.MethodPatch, "/patch"},
		{http.MethodHead, "/head"},
		{http.MethodOptions, "/options"},
		{http.MethodGet, "/any"},
		{http.MethodDelete, "/any"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		srv.Engine().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("%s %s should be 200: %d", tc.method, tc.path, w.Code)
		}
	}
}

// 验证 Use 全局中间件与 RouterGroup 方法糖。
func TestUseAndGroupMethodSugar(t *testing.T) {
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	srv.Use(func(c *gin.Context) {
		c.Header("X-Global", "on")
		c.Next()
	})
	v1 := srv.Group("/v1")
	v1.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
	v1.POST("/echo", func(c *gin.Context) { c.String(200, "echo") })
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/v1/ping", "pong"},
		{http.MethodPost, "/v1/echo", "echo"},
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		srv.Engine().ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Body.String() != tc.body {
			t.Fatalf("%s %s should be 200 %q: %d %q", tc.method, tc.path, tc.body, w.Code, w.Body.String())
		}
		if w.Header().Get("X-Global") != "on" {
			t.Fatalf("global middleware should apply to %s %s", tc.method, tc.path)
		}
	}
}

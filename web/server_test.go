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
	srv := New(cfg)
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
	srv := New(cfg)
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
	srv := New(cfg)
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

// 验证超时配置：零值回落默认值，显式配置生效。
func TestTimeoutConfig(t *testing.T) {
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	srv := New(cfg)
	httpSrv := srv.newHTTPServer("127.0.0.1", 8080)
	if httpSrv.ReadHeaderTimeout != 10*time.Second || httpSrv.IdleTimeout != 120*time.Second {
		t.Fatalf("unexpected default timeouts: header=%v idle=%v", httpSrv.ReadHeaderTimeout, httpSrv.IdleTimeout)
	}
	cfg.ReadHeaderTimeout = 3 * time.Second
	cfg.IdleTimeout = 60 * time.Second
	srv = New(cfg)
	httpSrv = srv.newHTTPServer("127.0.0.1", 8080)
	if httpSrv.ReadHeaderTimeout != 3*time.Second || httpSrv.IdleTimeout != 60*time.Second {
		t.Fatalf("explicit timeouts should be respected: header=%v idle=%v", httpSrv.ReadHeaderTimeout, httpSrv.IdleTimeout)
	}
	// 零值配置也应回落默认值（而非无超时）
	srv = New(conf.HTTPConfig{Port: -1})
	httpSrv = srv.newHTTPServer("127.0.0.1", 8080)
	if httpSrv.ReadHeaderTimeout <= 0 || httpSrv.ReadTimeout <= 0 || httpSrv.WriteTimeout <= 0 || httpSrv.IdleTimeout <= 0 {
		t.Fatal("zero config should fall back to safe defaults")
	}
}

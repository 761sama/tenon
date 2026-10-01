package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon/conf"
)

// 验证请求 ID 中间件：请求携带时以请求为准，缺失时生成并写入 context 与响应头。
func TestRequestIDMiddleware(t *testing.T) {
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	srv.Use(RequestIDMiddleware())
	srv.GET("/ping", func(c *gin.Context) {
		c.String(200, c.GetString(RequestIDKey))
	})
	// 请求已携带：以请求为准
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set(RequestIDHeader, "req-123")
	srv.Engine().ServeHTTP(w, req)
	if w.Body.String() != "req-123" || w.Header().Get(RequestIDHeader) != "req-123" {
		t.Fatalf("request-carried id should win: body=%q header=%q", w.Body.String(), w.Header().Get(RequestIDHeader))
	}
	// 缺失：生成并写入 context 与响应头
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/ping", nil)
	srv.Engine().ServeHTTP(w, req)
	id := w.Body.String()
	if id == "" || w.Header().Get(RequestIDHeader) != id {
		t.Fatalf("missing id should be generated: body=%q header=%q", id, w.Header().Get(RequestIDHeader))
	}
	// 再次请求生成不同的 ID
	w2 := httptest.NewRecorder()
	srv.Engine().ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if w2.Body.String() == id {
		t.Fatal("generated ids should be unique")
	}
}

// 验证非法客户端请求 ID 不回显：长度超限或含非法字符时重新生成。
func TestRequestIDMiddlewareRejectsInvalid(t *testing.T) {
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	srv.Use(RequestIDMiddleware())
	srv.GET("/ping", func(c *gin.Context) {
		c.String(200, c.GetString(RequestIDKey))
	})
	// 非法输入：超长、含空格、含斜杠、含中文、含控制字符、含引号
	invalid := []string{
		strings.Repeat("a", 65),
		"req 123",
		"req/123",
		"请求-123",
		"req\t123",
		`req"123`,
	}
	for _, bad := range invalid {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set(RequestIDHeader, bad)
		srv.Engine().ServeHTTP(w, req)
		got := w.Body.String()
		if got == bad || w.Header().Get(RequestIDHeader) == bad {
			t.Fatalf("invalid id %q should not be echoed: body=%q", bad, got)
		}
		if !validRequestID(got) {
			t.Fatalf("fallback id should be valid: %q", got)
		}
	}
	// 合法边界：恰好 64 字符与全部允许字符
	valid := []string{strings.Repeat("a", 64), "Abc-123_X.Y"}
	for _, good := range valid {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set(RequestIDHeader, good)
		srv.Engine().ServeHTTP(w, req)
		if w.Body.String() != good {
			t.Fatalf("valid id %q should be echoed: %q", good, w.Body.String())
		}
	}
}

// 验证 Bind：按 Content-Type 绑定，失败仅返回错误不写响应。
func TestBind(t *testing.T) {
	type form struct {
		Name string `json:"name" form:"name"`
	}
	cfg := conf.DefaultHTTPConfig()
	cfg.Port = -1
	srv, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %s", err)
	}
	srv.POST("/bind", func(c *gin.Context) {
		var f form
		if err := Bind(c, &f); err != nil {
			c.String(http.StatusBadRequest, "bind failed")
			return
		}
		c.String(200, f.Name)
	})
	// JSON 绑定成功
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader(`{"name":"tenon"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.Engine().ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "tenon" {
		t.Fatalf("json bind should succeed: %d %q", w.Code, w.Body.String())
	}
	// 表单绑定成功
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader("name=form-name"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.Engine().ServeHTTP(w, req)
	if w.Code != 200 || w.Body.String() != "form-name" {
		t.Fatalf("form bind should succeed: %d %q", w.Code, w.Body.String())
	}
	// 非法 JSON：仅返回错误，响应由调用方写
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/bind", strings.NewReader(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	srv.Engine().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest || w.Body.String() != "bind failed" {
		t.Fatalf("bind failure should be decided by caller: %d %q", w.Code, w.Body.String())
	}
}

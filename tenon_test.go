package tenon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 验证中间件注册与按名称获取。
func TestMiddlewareRegistry(t *testing.T) {
	RegMiddleware("test-mark", func(c *gin.Context) {
		c.Header("X-Mark", "ok")
		c.Next()
	})
	if !HasMiddleware("test-mark") {
		t.Fatal("middleware should be registered")
	}
	if Middleware("test-mark") == nil {
		t.Fatal("middleware should not be nil")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("middleware not registered should panic")
		}
	}()
	Middleware("not-exists")
}

// 验证路由注册、中间件执行与统一响应。
func TestRouterAndResponse(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HTTP.Port = -1
	srv := WebServer(cfg)
	RegMiddleware("test-auth", func(c *gin.Context) {
		c.Set("authed", true)
		c.Next()
	})
	srv.Router("GET", "/ping", Middleware("test-auth"), func(c *gin.Context) {
		if v, _ := c.Get("authed"); v != true {
			t.Error("middleware should run before controller")
		}
		Success(c, gin.H{"msg": "pong"})
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	srv.Engine().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	var resp struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Result  map[string]any `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %s", err)
	}
	if resp.Code != 0 || resp.Result["msg"] != "pong" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// 验证路由组前缀与嵌套分组。
func TestRouterGroup(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HTTP.Port = -1
	srv := WebServer(cfg)
	v1 := srv.Group("/v1")
	admin := v1.Group("/admin")
	admin.Router("GET", "/info", func(c *gin.Context) {
		Success(c, "admin-info")
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/info", nil)
	srv.Engine().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

// 验证无路由时返回统一错误响应。
func TestNoRoute(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HTTP.Port = -1
	srv := WebServer(cfg)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/no-such-path", nil)
	srv.Engine().ServeHTTP(w, req)
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %s", err)
	}
	if resp.Code != -1 {
		t.Fatalf("unexpected response code: %d", resp.Code)
	}
}

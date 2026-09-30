package tenon

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"gopkg.761sama.com/tenon/app"
)

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

// 轮询直到 URL 返回 200 或超时。
// 入参: url (待探测地址)
func waitHTTP200(t *testing.T, url string) {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not respond 200 in time: %s", url)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// 验证 Enable 为空时不初始化任何模块，主服务（HTTP）仍正常启动并返回 200。
func TestWebAppEmptyEnable(t *testing.T) {
	cfg := DefaultHTTPConfig()
	cfg.Address = "127.0.0.1"
	cfg.Port = freePort(t)
	customRan := false
	w := NewWebApp(WebAppOptions{
		HTTP:   cfg,
		Custom: []app.Module{{Name: "probe", Init: func() error { customRan = true; return nil }}},
	})
	if err := w.Init(); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	if customRan {
		t.Fatal("custom module should not be initialized when not enabled")
	}
	s := w.Server()
	s.Router("GET", "/ping", func(c *gin.Context) {
		Success(c, gin.H{"msg": "pong"})
	})
	if err := s.Start(); err != nil {
		t.Fatalf("start failed: %s", err)
	}
	defer w.Stop(2 * time.Second)
	waitHTTP200(t, fmt.Sprintf("http://127.0.0.1:%d/ping", cfg.Port))
}

// 验证 Enable 只含 log 时服务正常启动（/ping 200），且未启用的模块不参与。
func TestWebAppEnableLogOnly(t *testing.T) {
	cfg := DefaultHTTPConfig()
	cfg.Address = "127.0.0.1"
	cfg.Port = freePort(t)
	w := NewWebApp(WebAppOptions{
		HTTP:   cfg,
		Log:    LogConfig{Enable: false},
		Enable: []string{ModuleLog},
	})
	s := w.Server()
	s.Router("GET", "/ping", func(c *gin.Context) {
		Success(c, gin.H{"msg": "pong"})
	})
	if err := w.Init(); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	if err := s.Start(); err != nil {
		t.Fatalf("start failed: %s", err)
	}
	defer w.Stop(2 * time.Second)
	waitHTTP200(t, fmt.Sprintf("http://127.0.0.1:%d/ping", cfg.Port))
	if w.Server().IsRunning() != true {
		t.Fatal("server should be running")
	}
}

// 验证 Enable 含未注册名时 Init 返回错误，且错误信息列出全部已知模块名。
func TestWebAppEnableUnknown(t *testing.T) {
	cfg := DefaultHTTPConfig()
	cfg.Port = -1
	w := NewWebApp(WebAppOptions{HTTP: cfg, Enable: []string{"ghost"}})
	err := w.Init()
	if err == nil {
		t.Fatal("unknown module should return error")
	}
	for _, name := range []string{`"ghost"`, ModuleLog, ModuleDatabase, ModuleRedis} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("error should contain %s: %s", name, err)
		}
	}
}

// 验证 Enable 含重复名时 Init 返回错误。
func TestWebAppEnableDuplicate(t *testing.T) {
	cfg := DefaultHTTPConfig()
	cfg.Port = -1
	w := NewWebApp(WebAppOptions{HTTP: cfg, Enable: []string{ModuleLog, ModuleLog}})
	if err := w.Init(); err == nil {
		t.Fatal("duplicate module in enable list should return error")
	}
}

// 验证自定义模块经 Custom 注册后可按名启用，且模块初始化时主服务尚未启动。
func TestWebAppCustomModule(t *testing.T) {
	cfg := DefaultHTTPConfig()
	cfg.Port = -1
	w := NewWebApp(WebAppOptions{HTTP: cfg})
	var events []string
	w2 := NewWebApp(WebAppOptions{
		HTTP: cfg,
		Custom: []app.Module{{
			Name: "probe",
			Init: func() error {
				events = append(events, "init:probe")
				if w.Server() != nil && w.Server().IsRunning() {
					events = append(events, "main-already-running")
				}
				return nil
			},
		}},
		Enable: []string{"probe"},
	})
	if err := w2.Init(); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	if len(events) != 1 || events[0] != "init:probe" {
		t.Fatalf("custom module should run before main service starts: %v", events)
	}
	w2.Stop(time.Second)
}

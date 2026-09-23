package conf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type appConfig struct {
	Version string `json:"version"`
	Port    int    `json:"port"`
	Name    string `json:"name"`
}

func (c *appConfig) SetVersion(v string) { c.Version = v }

func defaultAppConfig() *appConfig {
	return &appConfig{Port: 8080, Name: "tenon"}
}

// 验证配置文件不存在时按默认值生成。
func TestLoadCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conf", "config.json")
	cfg, from, err := Load(path, defaultAppConfig, "1.0.0")
	if err != nil {
		t.Fatalf("load failed: %s", err)
	}
	if from != FromCreate {
		t.Fatalf("unexpected from: %s", from)
	}
	if cfg.Port != 8080 || cfg.Version != "1.0.0" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config file should be created: %s", err)
	}
	var onDisk appConfig
	if err := json.Unmarshal(body, &onDisk); err != nil || onDisk.Port != 8080 {
		t.Fatalf("unexpected file content: %v, %+v", err, onDisk)
	}
	info, _ := os.Stat(path)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected file perm: %v", info.Mode().Perm())
	}
}

// 验证已存在配置的加载、字段补全与版本回写。
func TestLoadMergeAndRewrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	// 旧版本配置：缺少 name 字段
	if err := os.WriteFile(path, []byte(`{"version":"0.0.1","port":9090}`), 0o600); err != nil {
		t.Fatalf("failed to seed config: %s", err)
	}
	cfg, from, err := Load(path, defaultAppConfig, "1.0.0")
	if err != nil {
		t.Fatalf("load failed: %s", err)
	}
	if from != FromLoad {
		t.Fatalf("unexpected from: %s", from)
	}
	if cfg.Port != 9090 {
		t.Fatalf("file value should override default: %d", cfg.Port)
	}
	if cfg.Name != "tenon" {
		t.Fatalf("missing field should be completed from defaults: %q", cfg.Name)
	}
	if cfg.Version != "1.0.0" {
		t.Fatalf("version should be rewritten: %q", cfg.Version)
	}
	body, _ := os.ReadFile(path)
	var onDisk appConfig
	if err := json.Unmarshal(body, &onDisk); err != nil {
		t.Fatalf("failed to parse rewritten file: %s", err)
	}
	if onDisk.Name != "tenon" || onDisk.Version != "1.0.0" {
		t.Fatalf("rewritten file should be completed: %+v", onDisk)
	}
}

// 验证损坏配置返回解析错误。
func TestLoadBroken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{broken`), 0o600); err != nil {
		t.Fatalf("failed to seed config: %s", err)
	}
	if _, _, err := Load(path, defaultAppConfig, "1.0.0"); err == nil {
		t.Fatal("broken config should return error")
	}
}

// 验证数据目录路径解析。
func TestResolveDataPath(t *testing.T) {
	if got := ResolveDataPath("data", "log/app.log"); got != filepath.Join("data", "log", "app.log") {
		t.Fatalf("unexpected path: %s", got)
	}
	abs := filepath.Join(string(filepath.Separator), "tmp", "x.log")
	if !filepath.IsAbs(abs) {
		abs, _ = filepath.Abs("x.log")
	}
	if got := ResolveDataPath("data", abs); got != filepath.Clean(abs) {
		t.Fatalf("abs path should be kept: %s", got)
	}
	if got := ResolveDataPath("", "a.db"); got != "a.db" {
		t.Fatalf("empty data dir should resolve to cwd-relative: %s", got)
	}
}

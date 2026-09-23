package conf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// 配置来源。
const (
	FromCreate = "create" // 配置文件不存在，已按默认值生成
	FromLoad   = "load"   // 配置文件已存在，已加载合并
)

// Versioned 配置版本回写接口：配置结构实现后，Load 加载时自动回写版本号。
type Versioned interface {
	SetVersion(v string)
}

// 通用配置文件加载（泛型，与具体配置结构解耦）：
// 文件不存在时以默认值生成；文件存在时先以默认值填充再反序列化覆盖（字段自动补全），
// 随后回写文件（补齐新增字段、回写版本号）。
// 入参: path (配置文件路径), defaults (默认配置构造函数), version (配置版本，空字符串跳过版本回写)
// 出参: 配置对象, 来源 (FromCreate/FromLoad), 错误
func Load[T any](path string, defaults func() *T, version string) (*T, string, error) {
	cfg := defaults()
	applyVersion := func() {
		if version != "" {
			if v, ok := any(cfg).(Versioned); ok {
				v.SetVersion(version)
			}
		}
	}
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		applyVersion()
		if err := writeFile(path, cfg); err != nil {
			return nil, "", err
		}
		return cfg, FromCreate, nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to stat config file: %w", err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read config file: %w", err)
	}
	if err := json.Unmarshal(body, cfg); err != nil {
		return nil, "", fmt.Errorf("failed to parse config file: %w", err)
	}
	applyVersion()
	if err := writeFile(path, cfg); err != nil {
		return nil, "", err
	}
	return cfg, FromLoad, nil
}

// 将配置以缩进 JSON 写入文件（0600 权限，自动创建父目录）。
// 入参: path (配置文件路径), cfg (配置对象)
// 出参: 写入错误
func writeFile[T any](path string, cfg *T) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

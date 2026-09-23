package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"

	"gopkg.761sama.com/tenon/bootstrap"
	"gopkg.761sama.com/tenon/conf"
)

var (
	// ErrRedisUnavailable redis 不可用哨兵错误：客户端为 nil 时由各操作返回，供调用方降级处理
	ErrRedisUnavailable = errors.New("redis unavailable")
	// ErrRedisNotConfigured redis 未配置哨兵错误：地址为空
	ErrRedisNotConfigured = errors.New("redis not configured")
)

// DefaultName 默认实例名称。
const DefaultName = "default"

// Instance Redis 实例：持有独立的客户端连接，全部操作方法挂在实例上。
type Instance struct {
	client *goredis.Client
}

var (
	mu        sync.RWMutex
	instances = map[string]*Instance{} // 实例注册表（名称 -> 实例）
)

// 显式初始化默认 Redis 实例。
// 入参: cfg (Redis 配置)
// 出参: 未配置或连接失败时返回错误
func Init(cfg conf.RedisConfig) error {
	return InitNamed(DefaultName, cfg)
}

// 显式初始化命名 Redis 实例，同名重复初始化会覆盖旧实例（旧连接随之关闭）。
// 入参: name (实例名称), cfg (Redis 配置)
// 出参: 未配置或连接失败时返回错误
func InitNamed(name string, cfg conf.RedisConfig) error {
	if cfg.Host == "" || cfg.Port == 0 {
		return fmt.Errorf("%w: redis address is empty", ErrRedisNotConfigured)
	}
	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     address,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		_ = rdb.Close()
		return fmt.Errorf("failed to connect redis @ %s: %w", address, err)
	}
	mu.Lock()
	if old, ok := instances[name]; ok {
		_ = old.client.Close()
	}
	instances[name] = &Instance{client: rdb}
	mu.Unlock()
	bootstrap.RegisterRelease("redis", CloseAll)
	log.Infof("redis instance %q connected @ %s/%d", name, address, cfg.DB)
	return nil
}

// 获取命名实例，不存在时返回 nil。
// 入参: name (实例名称)
// 出参: Redis 实例
func Named(name string) *Instance {
	mu.RLock()
	defer mu.RUnlock()
	return instances[name]
}

// 获取默认实例，未初始化时返回持 nil 客户端的空实例（各操作返回 ErrRedisUnavailable）。
// 出参: 默认 Redis 实例
func Default() *Instance {
	if ins := Named(DefaultName); ins != nil {
		return ins
	}
	return &Instance{}
}

// 默认实例是否可用。
// 出参: 默认实例是否已初始化
func IsAvailable() bool {
	return Default().IsAvailable()
}

// 获取默认实例的底层 go-redis 客户端，未初始化时返回 nil。
// 出参: go-redis 客户端
func Client() *goredis.Client {
	return Default().Client()
}

// 关闭全部 Redis 实例连接。
func CloseAll() {
	mu.Lock()
	defer mu.Unlock()
	for name, ins := range instances {
		if err := ins.client.Close(); err != nil {
			log.Errorf("failed to close redis instance %q: %s", name, err.Error())
		}
	}
	instances = map[string]*Instance{}
	log.Info("all redis connections closed")
}

// 构建 redis 键：以冒号拼接键片段。
// 入参: parts (键片段)
// 出参: 拼接后的完整键
func BuildKey(parts ...string) string {
	return strings.Join(parts, ":")
}

// 实例是否可用。
// 出参: 客户端是否已初始化
func (i *Instance) IsAvailable() bool {
	return i.client != nil
}

// 获取实例的底层 go-redis 客户端，未初始化时返回 nil。
// 出参: go-redis 客户端
func (i *Instance) Client() *goredis.Client {
	return i.client
}

// 关闭实例连接。
func (i *Instance) Close() {
	if i.client == nil {
		return
	}
	if err := i.client.Close(); err != nil {
		log.Errorf("failed to close redis: %s", err.Error())
	}
	i.client = nil
}

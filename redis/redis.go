package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

var client *goredis.Client

// 显式初始化 Redis 客户端并测试连通性；初始化成功后注册停机释放。
// 入参: cfg (Redis 配置)
// 出参: 未配置或连接失败时返回错误
func Init(cfg conf.RedisConfig) error {
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
	client = rdb
	bootstrap.RegisterRelease("redis", Close)
	log.Infof("redis connected @ %s/%d", address, cfg.DB)
	return nil
}

// Redis 是否可用。
// 出参: 客户端是否已初始化
func IsAvailable() bool {
	return client != nil
}

// 获取底层 go-redis 客户端，未初始化时返回 nil。
// 出参: go-redis 客户端
func Client() *goredis.Client {
	return client
}

// 关闭 Redis 连接。
func Close() {
	if client == nil {
		return
	}
	log.Info("closing redis connection")
	if err := client.Close(); err != nil {
		log.Errorf("failed to close redis: %s", err.Error())
	}
	client = nil
}

// 构建 redis 键：以冒号拼接键片段。
// 入参: parts (键片段)
// 出参: 拼接后的完整键
func BuildKey(parts ...string) string {
	return strings.Join(parts, ":")
}

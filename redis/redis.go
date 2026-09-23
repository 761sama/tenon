package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"

	"gopkg.761sama.com/tenon/bootstrap"
	"gopkg.761sama.com/tenon/conf"
)

var client *goredis.Client

func init() {
	bootstrap.RegisterInitModule("redis", Init)
	bootstrap.RegisterRelease("redis", Close)
}

// 初始化 Redis 模块（由 bootstrap 调用），Enable 为 false 时跳过。
// 入参: cfg (总配置)
func Init(cfg conf.Config) {
	if !cfg.Redis.Enable {
		return
	}
	client = goredis.NewClient(&goredis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect redis: %s", err.Error())
	}
	log.Infof("init redis @ %s:%d/%d", cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB)
}

// 获取 Redis 客户端，未初始化时返回 nil。
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

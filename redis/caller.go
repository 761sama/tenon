package redis

import (
	"time"

	goredis "github.com/redis/go-redis/v9"

	"gopkg.761sama.com/tenon/conf"
)

// Ops Redis 操作门面：方法作用于显式初始化后的全局客户端，
// 客户端不可用时各方法返回 ErrRedisUnavailable，供调用方降级处理。
type Ops struct{}

// 显式初始化 Redis 客户端并测试连通性。
// 入参: cfg (Redis 配置)
// 出参: 未配置或连接失败时返回错误
func (Ops) Init(cfg conf.RedisConfig) error { return Init(cfg) }

// 关闭 Redis 连接。
func (Ops) Close() { Close() }

// Redis 是否可用。
// 出参: 客户端是否已初始化
func (Ops) IsAvailable() bool { return IsAvailable() }

// 获取底层 go-redis 客户端，未初始化时返回 nil。
// 出参: go-redis 客户端
func (Ops) Client() *goredis.Client { return Client() }

// 构建 redis 键：以冒号拼接键片段。
// 入参: parts (键片段)
// 出参: 拼接后的完整键
func (Ops) BuildKey(parts ...string) string { return BuildKey(parts...) }

// ==================== 字符串操作 ====================

// 字符串读，键不存在时返回空串与 nil 错误。
func (Ops) Get(key ...string) (string, error) { return get(client, key...) }

// 字符串写。
func (Ops) Set(value string, key ...string) error { return set(client, value, key...) }

// 字符串写入并设置过期时间。
func (Ops) SetWithTTL(value string, ttl time.Duration, key ...string) error {
	return setWithTTL(client, value, ttl, key...)
}

// ==================== 列表操作 ====================

// 列表范围读取。
func (Ops) LGet(start, stop int64, key ...string) ([]string, error) { return lget(client, start, stop, key...) }

// 列表全部读取。
func (Ops) LGetAll(key ...string) ([]string, error) { return lgetall(client, key...) }

// 列表左弹出。
func (Ops) LLPop(key ...string) (string, error) { return llpop(client, key...) }

// 列表右弹出。
func (Ops) LRPop(key ...string) (string, error) { return lrpop(client, key...) }

// 列表左推入。
func (Ops) LLPush(values []string, key ...string) error { return llpush(client, values, key...) }

// 列表右推入。
func (Ops) LRPush(values []string, key ...string) error { return lrpush(client, values, key...) }

// 列表长度。
func (Ops) LLen(key ...string) (int64, error) { return llen(client, key...) }

// 列表截断。
func (Ops) LTrim(start, stop int64, key ...string) error { return ltrim(client, start, stop, key...) }

// ==================== 集合操作 ====================

// 集合添加。
func (Ops) SAdd(values []string, key ...string) error { return sadd(client, values, key...) }

// 集合获取全部。
func (Ops) SGet(key ...string) ([]string, error) { return sget(client, key...) }

// 集合判断存在。
func (Ops) SHas(member string, key ...string) (bool, error) { return shas(client, member, key...) }

// ==================== 有序集合操作 ====================

// 有序集合添加。
func (Ops) ZAdd(values []goredis.Z, key ...string) error { return zadd(client, values, key...) }

// 有序集合范围读取（含分数）。
func (Ops) ZGet(start, stop int64, key ...string) ([]goredis.Z, error) { return zget(client, start, stop, key...) }

// 有序集合全部读取（含分数）。
func (Ops) ZGetAll(key ...string) ([]goredis.Z, error) { return zgetall(client, key...) }

// 有序集合计数。
func (Ops) ZCount(key ...string) (int64, error) { return zcount(client, key...) }

// ==================== 哈希操作 ====================

// 哈希设置。
func (Ops) HSet(value any, key ...string) error { return hset(client, value, key...) }

// 哈希单个字段。
func (Ops) HGet(field string, key ...string) (any, error) { return hget(client, field, key...) }

// 哈希全部字段。
func (Ops) HGetAll(key ...string) (map[string]any, error) { return hgetall(client, key...) }

// 哈希字段自增。
func (Ops) HIncrBy(field string, increment int64, key ...string) (int64, error) {
	return hincrby(client, field, increment, key...)
}

// ==================== 通用操作 ====================

// 删除键。
func (Ops) Del(key ...string) error { return del(client, key...) }

// 判断键是否存在。
func (Ops) Exists(key ...string) (bool, error) { return exists(client, key...) }

// 键值自增 1。
func (Ops) Incr(key ...string) (int64, error) { return incr(client, key...) }

// 键值增加指定值。
func (Ops) IncrBy(value int64, key ...string) (int64, error) { return incrBy(client, value, key...) }

// 匹配键。
func (Ops) Keys(pattern ...string) ([]string, error) { return keys(client, pattern...) }

// 设置过期时间。
func (Ops) SetTTL(ttl time.Duration, key ...string) error { return setTTL(client, ttl, key...) }

// 获取过期时间。
func (Ops) GetTTL(key ...string) (time.Duration, error) { return getTTL(client, key...) }

// 执行 Lua 脚本。
func (Ops) Eval(script string, args []any, key ...string) (any, error) { return eval(client, script, args, key...) }

// 原子轮换：旧键存在时写新键并删旧键，设置新键 TTL。
func (Ops) Rotate(data map[string]any, ttl time.Duration, oldKey, newKey []string) (bool, error) {
	return rotate(client, data, ttl, oldKey, newKey)
}

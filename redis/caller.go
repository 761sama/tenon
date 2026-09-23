package redis

import (
	"time"

	goredis "github.com/redis/go-redis/v9"

	"gopkg.761sama.com/tenon/conf"
)

// Ops Redis 默认实例门面：方法作用于显式初始化后的默认实例，
// 经根包暴露为 tenon.Redis 使用；命名实例通过 InitNamed + Named 获取。
type Ops struct{}

// 显式初始化默认实例并测试连通性。
// 入参: cfg (Redis 配置)
// 出参: 未配置或连接失败时返回错误
func (Ops) Init(cfg conf.RedisConfig) error { return Init(cfg) }

// 显式初始化命名实例（多实例）。
// 入参: name (实例名称), cfg (Redis 配置)
// 出参: 未配置或连接失败时返回错误
func (Ops) InitNamed(name string, cfg conf.RedisConfig) error { return InitNamed(name, cfg) }

// 获取命名实例，不存在时返回 nil。
// 入参: name (实例名称)
// 出参: Redis 实例
func (Ops) Named(name string) *Instance { return Named(name) }

// 关闭全部实例连接。
func (Ops) Close() { CloseAll() }

// 默认实例是否可用。
// 出参: 默认实例是否已初始化
func (Ops) IsAvailable() bool { return IsAvailable() }

// 获取默认实例的底层 go-redis 客户端，未初始化时返回 nil。
// 出参: go-redis 客户端
func (Ops) Client() *goredis.Client { return Client() }

// 构建 redis 键：以冒号拼接键片段。
// 入参: parts (键片段)
// 出参: 拼接后的完整键
func (Ops) BuildKey(parts ...string) string { return BuildKey(parts...) }

// ==================== 默认实例操作（委托给 Default 实例） ====================

// 字符串读，键不存在时返回空串与 nil 错误。
func (Ops) Get(key ...string) (string, error) { return Default().Get(key...) }

// 字符串写。
func (Ops) Set(value string, key ...string) error { return Default().Set(value, key...) }

// 字符串写入并设置过期时间。
func (Ops) SetWithTTL(value string, ttl time.Duration, key ...string) error {
	return Default().SetWithTTL(value, ttl, key...)
}

// 列表范围读取。
func (Ops) LGet(start, stop int64, key ...string) ([]string, error) { return Default().LGet(start, stop, key...) }

// 列表全部读取。
func (Ops) LGetAll(key ...string) ([]string, error) { return Default().LGetAll(key...) }

// 列表左弹出。
func (Ops) LLPop(key ...string) (string, error) { return Default().LLPop(key...) }

// 列表右弹出。
func (Ops) LRPop(key ...string) (string, error) { return Default().LRPop(key...) }

// 列表左推入。
func (Ops) LLPush(values []string, key ...string) error { return Default().LLPush(values, key...) }

// 列表右推入。
func (Ops) LRPush(values []string, key ...string) error { return Default().LRPush(values, key...) }

// 列表长度。
func (Ops) LLen(key ...string) (int64, error) { return Default().LLen(key...) }

// 列表截断。
func (Ops) LTrim(start, stop int64, key ...string) error { return Default().LTrim(start, stop, key...) }

// 集合添加。
func (Ops) SAdd(values []string, key ...string) error { return Default().SAdd(values, key...) }

// 集合获取全部。
func (Ops) SGet(key ...string) ([]string, error) { return Default().SGet(key...) }

// 集合判断存在。
func (Ops) SHas(member string, key ...string) (bool, error) { return Default().SHas(member, key...) }

// 有序集合添加。
func (Ops) ZAdd(values []goredis.Z, key ...string) error { return Default().ZAdd(values, key...) }

// 有序集合范围读取（含分数）。
func (Ops) ZGet(start, stop int64, key ...string) ([]goredis.Z, error) { return Default().ZGet(start, stop, key...) }

// 有序集合全部读取（含分数）。
func (Ops) ZGetAll(key ...string) ([]goredis.Z, error) { return Default().ZGetAll(key...) }

// 有序集合计数。
func (Ops) ZCount(key ...string) (int64, error) { return Default().ZCount(key...) }

// 哈希设置。
func (Ops) HSet(value any, key ...string) error { return Default().HSet(value, key...) }

// 哈希单个字段。
func (Ops) HGet(field string, key ...string) (any, error) { return Default().HGet(field, key...) }

// 哈希全部字段。
func (Ops) HGetAll(key ...string) (map[string]any, error) { return Default().HGetAll(key...) }

// 哈希字段自增。
func (Ops) HIncrBy(field string, increment int64, key ...string) (int64, error) {
	return Default().HIncrBy(field, increment, key...)
}

// 删除键。
func (Ops) Del(key ...string) error { return Default().Del(key...) }

// 判断键是否存在。
func (Ops) Exists(key ...string) (bool, error) { return Default().Exists(key...) }

// 键值自增 1。
func (Ops) Incr(key ...string) (int64, error) { return Default().Incr(key...) }

// 键值增加指定值。
func (Ops) IncrBy(value int64, key ...string) (int64, error) { return Default().IncrBy(value, key...) }

// 匹配键。
func (Ops) Keys(pattern ...string) ([]string, error) { return Default().Keys(pattern...) }

// 设置过期时间。
func (Ops) SetTTL(ttl time.Duration, key ...string) error { return Default().SetTTL(ttl, key...) }

// 获取过期时间。
func (Ops) GetTTL(key ...string) (time.Duration, error) { return Default().GetTTL(key...) }

// 执行 Lua 脚本。
func (Ops) Eval(script string, args []any, key ...string) (any, error) {
	return Default().Eval(script, args, key...)
}

// 原子轮换：旧键存在时写新键并删旧键，设置新键 TTL。
func (Ops) Rotate(data map[string]any, ttl time.Duration, oldKey, newKey []string) (bool, error) {
	return Default().Rotate(data, ttl, oldKey, newKey)
}

// ==================== 实例操作 ====================

// 字符串读，键不存在时返回空串与 nil 错误。
func (i *Instance) Get(key ...string) (string, error) { return get(i.client, key...) }

// 字符串写。
func (i *Instance) Set(value string, key ...string) error { return set(i.client, value, key...) }

// 字符串写入并设置过期时间。
func (i *Instance) SetWithTTL(value string, ttl time.Duration, key ...string) error {
	return setWithTTL(i.client, value, ttl, key...)
}

// 列表范围读取。
func (i *Instance) LGet(start, stop int64, key ...string) ([]string, error) {
	return lget(i.client, start, stop, key...)
}

// 列表全部读取。
func (i *Instance) LGetAll(key ...string) ([]string, error) { return lgetall(i.client, key...) }

// 列表左弹出。
func (i *Instance) LLPop(key ...string) (string, error) { return llpop(i.client, key...) }

// 列表右弹出。
func (i *Instance) LRPop(key ...string) (string, error) { return lrpop(i.client, key...) }

// 列表左推入。
func (i *Instance) LLPush(values []string, key ...string) error { return llpush(i.client, values, key...) }

// 列表右推入。
func (i *Instance) LRPush(values []string, key ...string) error { return lrpush(i.client, values, key...) }

// 列表长度。
func (i *Instance) LLen(key ...string) (int64, error) { return llen(i.client, key...) }

// 列表截断。
func (i *Instance) LTrim(start, stop int64, key ...string) error { return ltrim(i.client, start, stop, key...) }

// 集合添加。
func (i *Instance) SAdd(values []string, key ...string) error { return sadd(i.client, values, key...) }

// 集合获取全部。
func (i *Instance) SGet(key ...string) ([]string, error) { return sget(i.client, key...) }

// 集合判断存在。
func (i *Instance) SHas(member string, key ...string) (bool, error) { return shas(i.client, member, key...) }

// 有序集合添加。
func (i *Instance) ZAdd(values []goredis.Z, key ...string) error { return zadd(i.client, values, key...) }

// 有序集合范围读取（含分数）。
func (i *Instance) ZGet(start, stop int64, key ...string) ([]goredis.Z, error) {
	return zget(i.client, start, stop, key...)
}

// 有序集合全部读取（含分数）。
func (i *Instance) ZGetAll(key ...string) ([]goredis.Z, error) { return zgetall(i.client, key...) }

// 有序集合计数。
func (i *Instance) ZCount(key ...string) (int64, error) { return zcount(i.client, key...) }

// 哈希设置。
func (i *Instance) HSet(value any, key ...string) error { return hset(i.client, value, key...) }

// 哈希单个字段。
func (i *Instance) HGet(field string, key ...string) (any, error) { return hget(i.client, field, key...) }

// 哈希全部字段。
func (i *Instance) HGetAll(key ...string) (map[string]any, error) { return hgetall(i.client, key...) }

// 哈希字段自增。
func (i *Instance) HIncrBy(field string, increment int64, key ...string) (int64, error) {
	return hincrby(i.client, field, increment, key...)
}

// 删除键。
func (i *Instance) Del(key ...string) error { return del(i.client, key...) }

// 判断键是否存在。
func (i *Instance) Exists(key ...string) (bool, error) { return exists(i.client, key...) }

// 键值自增 1。
func (i *Instance) Incr(key ...string) (int64, error) { return incr(i.client, key...) }

// 键值增加指定值。
func (i *Instance) IncrBy(value int64, key ...string) (int64, error) { return incrBy(i.client, value, key...) }

// 匹配键。
func (i *Instance) Keys(pattern ...string) ([]string, error) { return keys(i.client, pattern...) }

// 设置过期时间。
func (i *Instance) SetTTL(ttl time.Duration, key ...string) error { return setTTL(i.client, ttl, key...) }

// 获取过期时间。
func (i *Instance) GetTTL(key ...string) (time.Duration, error) { return getTTL(i.client, key...) }

// 执行 Lua 脚本。
func (i *Instance) Eval(script string, args []any, key ...string) (any, error) {
	return eval(i.client, script, args, key...)
}

// 原子轮换：旧键存在时写新键并删旧键，设置新键 TTL。
func (i *Instance) Rotate(data map[string]any, ttl time.Duration, oldKey, newKey []string) (bool, error) {
	return rotate(i.client, data, ttl, oldKey, newKey)
}

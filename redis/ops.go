package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var ctx = context.Background()

// 将 []string 转换为 []any 以适配 Redis 客户端可变参数。
// 入参: s (字符串切片)
// 出参: 对应的 any 切片
func s2i(s []string) []any {
	args := make([]any, len(s))
	for i, v := range s {
		args[i] = v
	}
	return args
}

// 写入字符串。
func set(r *goredis.Client, value string, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.Set(ctx, BuildKey(key...), value, 0).Err()
}

// 写入字符串并设置过期时间。
func setWithTTL(r *goredis.Client, value string, ttl time.Duration, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.Set(ctx, BuildKey(key...), value, ttl).Err()
}

// 获取字符串，键不存在时返回空串与 nil 错误。
func get(r *goredis.Client, key ...string) (string, error) {
	if r == nil {
		return "", ErrRedisUnavailable
	}
	val, err := r.Get(ctx, BuildKey(key...)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// 获取列表指定范围。
func lget(r *goredis.Client, start, stop int64, key ...string) ([]string, error) {
	if r == nil {
		return []string{}, ErrRedisUnavailable
	}
	val, err := r.LRange(ctx, BuildKey(key...), start, stop).Result()
	if err != nil {
		return []string{}, err
	}
	return val, nil
}

// 获取列表全部。
func lgetall(r *goredis.Client, key ...string) ([]string, error) {
	return lget(r, 0, -1, key...)
}

// 列表左弹出，键不存在时返回空串与 nil 错误。
func llpop(r *goredis.Client, key ...string) (string, error) {
	if r == nil {
		return "", ErrRedisUnavailable
	}
	val, err := r.LPop(ctx, BuildKey(key...)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// 列表右弹出，键不存在时返回空串与 nil 错误。
func lrpop(r *goredis.Client, key ...string) (string, error) {
	if r == nil {
		return "", ErrRedisUnavailable
	}
	val, err := r.RPop(ctx, BuildKey(key...)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// 列表左推入。
func llpush(r *goredis.Client, values []string, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.LPush(ctx, BuildKey(key...), s2i(values)...).Err()
}

// 列表右推入。
func lrpush(r *goredis.Client, values []string, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.RPush(ctx, BuildKey(key...), s2i(values)...).Err()
}

// 获取列表长度。
func llen(r *goredis.Client, key ...string) (int64, error) {
	if r == nil {
		return 0, ErrRedisUnavailable
	}
	return r.LLen(ctx, BuildKey(key...)).Result()
}

// 截断列表。
func ltrim(r *goredis.Client, start, stop int64, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.LTrim(ctx, BuildKey(key...), start, stop).Err()
}

// 向集合中添加元素。
func sadd(r *goredis.Client, values []string, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.SAdd(ctx, BuildKey(key...), s2i(values)...).Err()
}

// 获取集合全部元素。
func sget(r *goredis.Client, key ...string) ([]string, error) {
	if r == nil {
		return []string{}, ErrRedisUnavailable
	}
	val, err := r.SMembers(ctx, BuildKey(key...)).Result()
	if err != nil {
		return []string{}, err
	}
	return val, nil
}

// 判断元素是否在集合中。
func shas(r *goredis.Client, member string, key ...string) (bool, error) {
	if r == nil {
		return false, ErrRedisUnavailable
	}
	return r.SIsMember(ctx, BuildKey(key...), member).Result()
}

// 向有序集合中添加元素。
func zadd(r *goredis.Client, values []goredis.Z, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.ZAdd(ctx, BuildKey(key...), values...).Err()
}

// 获取有序集合指定范围的元素（含分数）。
func zget(r *goredis.Client, start, stop int64, key ...string) ([]goredis.Z, error) {
	if r == nil {
		return []goredis.Z{}, ErrRedisUnavailable
	}
	return r.ZRangeWithScores(ctx, BuildKey(key...), start, stop).Result()
}

// 获取有序集合全部元素（含分数）。
func zgetall(r *goredis.Client, key ...string) ([]goredis.Z, error) {
	return zget(r, 0, -1, key...)
}

// 获取有序集合元素个数。
func zcount(r *goredis.Client, key ...string) (int64, error) {
	if r == nil {
		return 0, ErrRedisUnavailable
	}
	return r.ZCard(ctx, BuildKey(key...)).Result()
}

// 向哈希中设置字段。
func hset(r *goredis.Client, value any, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.HSet(ctx, BuildKey(key...), value).Err()
}

// 获取哈希中所有字段。
func hgetall(r *goredis.Client, key ...string) (map[string]any, error) {
	if r == nil {
		return map[string]any{}, ErrRedisUnavailable
	}
	result, err := r.HGetAll(ctx, BuildKey(key...)).Result()
	if err != nil {
		return map[string]any{}, err
	}
	m := make(map[string]any, len(result))
	for k, v := range result {
		m[k] = v
	}
	return m, nil
}

// 获取哈希中单个字段，字段不存在时返回 nil 与 nil 错误。
func hget(r *goredis.Client, field string, key ...string) (any, error) {
	if r == nil {
		return nil, ErrRedisUnavailable
	}
	val, err := r.HGet(ctx, BuildKey(key...), field).Result()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return val, nil
}

// 哈希字段自增。
func hincrby(r *goredis.Client, field string, increment int64, key ...string) (int64, error) {
	if r == nil {
		return 0, ErrRedisUnavailable
	}
	return r.HIncrBy(ctx, BuildKey(key...), field, increment).Result()
}

// 删除键。
func del(r *goredis.Client, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.Del(ctx, BuildKey(key...)).Err()
}

// 设置键的过期时间。
func setTTL(r *goredis.Client, ttl time.Duration, key ...string) error {
	if r == nil {
		return ErrRedisUnavailable
	}
	return r.Expire(ctx, BuildKey(key...), ttl).Err()
}

// 获取键的过期时间。
func getTTL(r *goredis.Client, key ...string) (time.Duration, error) {
	if r == nil {
		return 0, ErrRedisUnavailable
	}
	return r.TTL(ctx, BuildKey(key...)).Result()
}

// 获取所有匹配的键。
func keys(r *goredis.Client, pattern ...string) ([]string, error) {
	if r == nil {
		return []string{}, ErrRedisUnavailable
	}
	return r.Keys(ctx, BuildKey(pattern...)).Result()
}

// 判断键是否存在。
func exists(r *goredis.Client, key ...string) (bool, error) {
	if r == nil {
		return false, ErrRedisUnavailable
	}
	n, err := r.Exists(ctx, BuildKey(key...)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// 键值自增 1。
func incr(r *goredis.Client, key ...string) (int64, error) {
	if r == nil {
		return 0, ErrRedisUnavailable
	}
	return r.Incr(ctx, BuildKey(key...)).Result()
}

// 键值增加指定值。
func incrBy(r *goredis.Client, value int64, key ...string) (int64, error) {
	if r == nil {
		return 0, ErrRedisUnavailable
	}
	return r.IncrBy(ctx, BuildKey(key...), value).Result()
}

// 执行 Lua 脚本，键片段拼接后作为单个 KEYS[1] 传入。
func eval(r *goredis.Client, script string, args []any, key ...string) (any, error) {
	if r == nil {
		return nil, ErrRedisUnavailable
	}
	return r.Eval(ctx, script, []string{BuildKey(key...)}, args...).Result()
}

// 原子轮换：旧键存在时写入新键数据并删除旧键，同时设置新键 TTL；
// 旧键不存在时不写入，用于保证并发下仅一个请求能成功消费旧键。
// 入参: r (redis 客户端), data (新键 hash 数据), ttl (新键过期时间), oldKey/newKey (键片段)
// 出参: 是否轮换成功（旧键不存在时返回 false）, 错误
func rotate(r *goredis.Client, data map[string]any, ttl time.Duration, oldKey, newKey []string) (bool, error) {
	if r == nil {
		return false, ErrRedisUnavailable
	}
	args := make([]any, 0, len(data)*2+1)
	for field, value := range data {
		args = append(args, field, value)
	}
	args = append(args, strconv.FormatInt(int64(ttl.Seconds()), 10))
	script := `
local oldKey = KEYS[1]
local newKey = KEYS[2]
if redis.call('EXISTS', oldKey) == 0 then
    return 0
end
local n = #ARGV - 1
redis.call('HSET', newKey, unpack(ARGV, 1, n))
redis.call('DEL', oldKey)
local ttl = tonumber(ARGV[#ARGV])
if ttl > 0 then
    redis.call('EXPIRE', newKey, ttl)
end
return 1`
	res, err := r.Eval(ctx, script, []string{BuildKey(oldKey...), BuildKey(newKey...)}, args...).Result()
	if err != nil {
		return false, err
	}
	n, ok := res.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected redis rotate result: %v", res)
	}
	return n == 1, nil
}

package redis

import (
	"errors"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"gopkg.761sama.com/tenon/conf"
)

const testPrefix = "tenon:test"

var ops = Ops{}

// 连接本地 Redis 完成初始化，不可用时跳过测试。
// 出参: 清理函数（删除测试键并关闭连接）
func setup(t *testing.T) func() {
	t.Helper()
	err := Init(conf.RedisConfig{Host: "127.0.0.1", Port: 6379})
	if err != nil {
		t.Skipf("local redis unavailable, skip: %s", err)
	}
	return func() {
		keys, _ := ops.Keys(testPrefix, "*")
		for _, k := range keys {
			_ = client.Del(ctx, k).Err()
		}
		Close()
	}
}

// 验证未初始化时操作返回 ErrRedisUnavailable。
func TestUnavailable(t *testing.T) {
	Close()
	if _, err := ops.Get("any"); !errors.Is(err, ErrRedisUnavailable) {
		t.Fatalf("expected ErrRedisUnavailable, got: %v", err)
	}
	if err := Init(conf.RedisConfig{}); !errors.Is(err, ErrRedisNotConfigured) {
		t.Fatalf("expected ErrRedisNotConfigured, got: %v", err)
	}
}

// 验证字符串读写、TTL 与删除。
func TestString(t *testing.T) {
	defer setup(t)()
	if err := ops.SetWithTTL("hello", time.Minute, testPrefix, "str"); err != nil {
		t.Fatalf("set failed: %s", err)
	}
	val, err := ops.Get(testPrefix, "str")
	if err != nil || val != "hello" {
		t.Fatalf("unexpected get: %q, %v", val, err)
	}
	ttl, err := ops.GetTTL(testPrefix, "str")
	if err != nil || ttl <= 0 {
		t.Fatalf("unexpected ttl: %v, %v", ttl, err)
	}
	if err := ops.Del(testPrefix, "str"); err != nil {
		t.Fatalf("del failed: %s", err)
	}
	exists, _ := ops.Exists(testPrefix, "str")
	if exists {
		t.Fatal("key should be deleted")
	}
	val, err = ops.Get(testPrefix, "str")
	if err != nil || val != "" {
		t.Fatalf("missing key should return empty: %q, %v", val, err)
	}
}

// 验证列表推入、读取、弹出与截断。
func TestList(t *testing.T) {
	defer setup(t)()
	if err := ops.LRPush([]string{"a", "b", "c"}, testPrefix, "list"); err != nil {
		t.Fatalf("rpush failed: %s", err)
	}
	n, _ := ops.LLen(testPrefix, "list")
	if n != 3 {
		t.Fatalf("unexpected len: %d", n)
	}
	all, _ := ops.LGetAll(testPrefix, "list")
	if len(all) != 3 || all[0] != "a" {
		t.Fatalf("unexpected list: %v", all)
	}
	if err := ops.LTrim(0, 1, testPrefix, "list"); err != nil {
		t.Fatalf("ltrim failed: %s", err)
	}
	v, _ := ops.LLPop(testPrefix, "list")
	if v != "a" {
		t.Fatalf("unexpected lpop: %q", v)
	}
	v, _ = ops.LRPop(testPrefix, "list")
	if v != "b" {
		t.Fatalf("unexpected lrpop: %q", v)
	}
}

// 验证集合添加、成员判断与获取。
func TestSet(t *testing.T) {
	defer setup(t)()
	if err := ops.SAdd([]string{"x", "y"}, testPrefix, "set"); err != nil {
		t.Fatalf("sadd failed: %s", err)
	}
	has, _ := ops.SHas("x", testPrefix, "set")
	if !has {
		t.Fatal("set should contain x")
	}
	members, _ := ops.SGet(testPrefix, "set")
	if len(members) != 2 {
		t.Fatalf("unexpected members: %v", members)
	}
}

// 验证有序集合添加、读取与计数。
func TestZSet(t *testing.T) {
	defer setup(t)()
	err := ops.ZAdd([]goredis.Z{{Score: 1, Member: "a"}, {Score: 2, Member: "b"}}, testPrefix, "zset")
	if err != nil {
		t.Fatalf("zadd failed: %s", err)
	}
	n, _ := ops.ZCount(testPrefix, "zset")
	if n != 2 {
		t.Fatalf("unexpected zcount: %d", n)
	}
	all, _ := ops.ZGetAll(testPrefix, "zset")
	if len(all) != 2 || all[0].Member != "a" || all[1].Score != 2 {
		t.Fatalf("unexpected zset: %v", all)
	}
}

// 验证哈希设置、读取、自增。
func TestHash(t *testing.T) {
	defer setup(t)()
	if err := ops.HSet(map[string]any{"name": "tenon", "age": 1}, testPrefix, "hash"); err != nil {
		t.Fatalf("hset failed: %s", err)
	}
	v, _ := ops.HGet("name", testPrefix, "hash")
	if v != "tenon" {
		t.Fatalf("unexpected hget: %v", v)
	}
	n, _ := ops.HIncrBy("age", 2, testPrefix, "hash")
	if n != 3 {
		t.Fatalf("unexpected hincrby: %d", n)
	}
	all, _ := ops.HGetAll(testPrefix, "hash")
	if all["name"] != "tenon" || all["age"] != "3" {
		t.Fatalf("unexpected hgetall: %v", all)
	}
}

// 验证自增、键匹配与 Lua 脚本。
func TestCommon(t *testing.T) {
	defer setup(t)()
	n, err := ops.Incr(testPrefix, "counter")
	if err != nil || n != 1 {
		t.Fatalf("unexpected incr: %d, %v", n, err)
	}
	n, _ = ops.IncrBy(9, testPrefix, "counter")
	if n != 10 {
		t.Fatalf("unexpected incrby: %d", n)
	}
	keys, _ := ops.Keys(testPrefix, "counter")
	if len(keys) != 1 {
		t.Fatalf("unexpected keys: %v", keys)
	}
	res, err := ops.Eval("return redis.call('GET', KEYS[1])", nil, testPrefix, "counter")
	if err != nil || res != "10" {
		t.Fatalf("unexpected eval: %v, %v", res, err)
	}
}

// 验证原子轮换：旧键存在时写新键删旧键，旧键不存在时返回 false。
func TestRotate(t *testing.T) {
	defer setup(t)()
	ok, err := ops.Rotate(map[string]any{"token": "v1"}, time.Minute, []string{testPrefix, "old"}, []string{testPrefix, "new"})
	if err != nil || ok {
		t.Fatalf("rotate without old key should fail: %v, %v", ok, err)
	}
	if err := ops.Set("old-token", testPrefix, "old"); err != nil {
		t.Fatalf("set failed: %s", err)
	}
	ok, err = ops.Rotate(map[string]any{"token": "v2"}, time.Minute, []string{testPrefix, "old"}, []string{testPrefix, "new"})
	if err != nil || !ok {
		t.Fatalf("rotate should succeed: %v, %v", ok, err)
	}
	oldExists, _ := ops.Exists(testPrefix, "old")
	if oldExists {
		t.Fatal("old key should be deleted")
	}
	v, _ := ops.HGet("token", testPrefix, "new")
	if v != "v2" {
		t.Fatalf("unexpected new key token: %v", v)
	}
	ttl, _ := ops.GetTTL(testPrefix, "new")
	if ttl <= 0 {
		t.Fatalf("new key should have ttl: %v", ttl)
	}
}

// 验证键构建。
func TestBuildKey(t *testing.T) {
	if got := BuildKey("a", "b", "c"); got != "a:b:c" {
		t.Fatalf("unexpected key: %s", got)
	}
}

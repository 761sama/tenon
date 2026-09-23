package bootstrap

import (
	"errors"
	"testing"
)

// 验证注册顺序执行与参数透传。
func TestInitOrderAndArgs(t *testing.T) {
	var order []string
	var gotArgs []string
	RegisterInitModule("test-a", func(args ...string) error {
		order = append(order, "a")
		gotArgs = args
		return nil
	})
	RegisterInitModule("test-b", func(args ...string) error {
		order = append(order, "b")
		return nil
	})
	if err := Init("x", "y"); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	if len(order) < 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("unexpected order: %v", order)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "x" {
		t.Fatalf("args should pass through: %v", gotArgs)
	}
	delete(RegisteredInitModules, "test-a")
	delete(RegisteredInitModules, "test-b")
	registerOrder = registerOrder[:len(registerOrder)-2]
}

// 验证错误传播：模块失败即中断，后续模块不执行。
func TestInitErrorPropagation(t *testing.T) {
	errBoom := errors.New("boom")
	ran := false
	RegisterInitModule("test-fail", func(args ...string) error { return errBoom })
	RegisterInitModule("test-after", func(args ...string) error { ran = true; return nil })
	err := Init()
	if !errors.Is(err, errBoom) {
		t.Fatalf("error should propagate: %v", err)
	}
	if ran {
		t.Fatal("modules after failure should not run")
	}
	delete(RegisteredInitModules, "test-fail")
	delete(RegisteredInitModules, "test-after")
	registerOrder = registerOrder[:len(registerOrder)-2]
}

// 验证 TinyInit：模块不存在与执行失败均返回错误。
func TestTinyInit(t *testing.T) {
	if err := TinyInit("not-exists"); err == nil {
		t.Fatal("unknown module should return error")
	}
	errBoom := errors.New("boom")
	RegisterInitModule("test-tiny", func(args ...string) error { return errBoom })
	if err := TinyInit("test-tiny"); !errors.Is(err, errBoom) {
		t.Fatalf("tiny init should propagate error: %v", err)
	}
	delete(RegisteredInitModules, "test-tiny")
	registerOrder = registerOrder[:len(registerOrder)-1]
}

// 验证资源释放按注册逆序执行。
func TestReleaseOrder(t *testing.T) {
	var order []string
	RegisterRelease("test-r1", func() { order = append(order, "r1") })
	RegisterRelease("test-r2", func() { order = append(order, "r2") })
	Release()
	n := len(order)
	if n < 2 || order[n-1] != "r1" || order[n-2] != "r2" {
		t.Fatalf("release should run in reverse order: %v", order)
	}
}

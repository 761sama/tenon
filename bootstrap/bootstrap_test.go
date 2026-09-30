package bootstrap

import (
	"testing"
	"time"
)

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

// 验证释放回调内调用 RegisterRelease 不会死锁（快照后执行，不持锁回调）。
func TestReleaseCallbackReentrant(t *testing.T) {
	done := make(chan struct{})
	RegisterRelease("test-reentrant", func() {
		RegisterRelease("test-reentrant-inner", func() {})
		close(done)
	})
	go Release()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock: RegisterRelease inside release callback blocked")
	}
	delete(registeredReleases, "test-reentrant")
	delete(registeredReleases, "test-reentrant-inner")
	releaseOrder = releaseOrder[:len(releaseOrder)-2]
}

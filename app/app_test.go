package app

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// 记录事件的假模块构造器：init/release 时向 events 追加记录。
// 入参: events (事件记录指针), name (模块名), initErr (初始化返回的错误)
// 出参: 模块定义
func recordModule(events *[]string, name string, initErr error) Module {
	return Module{
		Name: name,
		Init: func() error {
			*events = append(*events, "init:"+name)
			return initErr
		},
		Release: func() {
			*events = append(*events, "release:"+name)
		},
	}
}

// 验证 Enable 为空时 Init 不初始化任何模块。
func TestEnableEmpty(t *testing.T) {
	var events []string
	a := New("test")
	a.Add(recordModule(&events, "m1", nil))
	a.Add(recordModule(&events, "m2", nil))
	if err := a.Init(); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	if len(events) != 0 {
		t.Fatalf("no module should be initialized with empty enable list: %v", events)
	}
	if len(a.Enabled()) != 0 {
		t.Fatalf("enabled list should be empty: %v", a.Enabled())
	}
}

// 验证加载顺序严格按 Enable 清单顺序（而非注册顺序）。
func TestEnableOrder(t *testing.T) {
	var events []string
	a := New("test")
	a.Add(recordModule(&events, "m1", nil))
	a.Add(recordModule(&events, "m2", nil))
	a.Add(recordModule(&events, "m3", nil))
	if err := a.Enable("m3", "m1"); err != nil {
		t.Fatalf("enable failed: %s", err)
	}
	if err := a.Init(); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	if len(events) != 2 || events[0] != "init:m3" || events[1] != "init:m1" {
		t.Fatalf("init should follow enable list order: %v", events)
	}
	got := a.Enabled()
	if len(got) != 2 || got[0] != "m3" || got[1] != "m1" {
		t.Fatalf("unexpected enabled list: %v", got)
	}
}

// 验证 Enable 含未注册名时返回错误，且错误信息列出全部已知模块名。
func TestEnableUnknown(t *testing.T) {
	a := New("test")
	a.Add(Module{Name: "m1", Init: func() error { return nil }})
	err := a.Enable("m1", "ghost")
	if err == nil {
		t.Fatal("unknown module should return error")
	}
	if !strings.Contains(err.Error(), `"ghost"`) || !strings.Contains(err.Error(), "m1") {
		t.Fatalf("error should name the unknown module and list known modules: %s", err)
	}
}

// 验证 Enable 含重复名时返回错误。
func TestEnableDuplicate(t *testing.T) {
	a := New("test")
	a.Add(Module{Name: "m1", Init: func() error { return nil }})
	if err := a.Enable("m1", "m1"); err == nil {
		t.Fatal("duplicate module in enable list should return error")
	}
}

// 验证模块 Init 失败时已成功初始化的模块按逆序释放，且后续模块不再执行。
func TestInitFailureRollback(t *testing.T) {
	var events []string
	a := New("test")
	a.Add(recordModule(&events, "m1", nil))
	a.Add(recordModule(&events, "m2", errors.New("boom")))
	a.Add(recordModule(&events, "m3", nil))
	if err := a.Enable("m1", "m2", "m3"); err != nil {
		t.Fatalf("enable failed: %s", err)
	}
	err := a.Init()
	if err == nil || !strings.Contains(err.Error(), `"m2"`) {
		t.Fatalf("init should fail with module name: %v", err)
	}
	want := []string{"init:m1", "init:m2", "release:m1"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("succeeded modules should be released in reverse order: %v", events)
	}
}

// 验证运行体在模块初始化完成之后才启动。
func TestRunBodyStartsAfterModuleInit(t *testing.T) {
	var events []string
	a := New("test")
	a.Add(recordModule(&events, "m1", nil))
	a.Add(recordModule(&events, "m2", nil))
	if err := a.Enable("m1", "m2"); err != nil {
		t.Fatalf("enable failed: %s", err)
	}
	a.OnRun(func() error {
		events = append(events, "run:main")
		return nil
	})
	if err := a.Init(); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	a.mu.Lock()
	runs := append([]func() error(nil), a.onRun...)
	a.mu.Unlock()
	for _, fn := range runs {
		if err := fn(); err != nil {
			t.Fatalf("run failed: %s", err)
		}
	}
	want := []string{"init:m1", "init:m2", "run:main"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("run body should start after all modules initialized: %v", events)
	}
}

// 验证 Add 同名重复注册覆盖实现且保留原注册顺序位置。
func TestAddOverrideKeepsOrder(t *testing.T) {
	a := New("test")
	a.Add(Module{Name: "m1", Init: func() error { return nil }})
	a.Add(Module{Name: "m2", Init: func() error { return nil }})
	replaced := false
	a.Add(Module{Name: "m1", Init: func() error { replaced = true; return nil }})
	names := a.Names()
	if len(names) != 2 || names[0] != "m1" || names[1] != "m2" {
		t.Fatalf("override should keep registration order: %v", names)
	}
	if !a.Has("m1") || a.Has("ghost") {
		t.Fatal("Has should reflect registered modules")
	}
	if err := a.Enable("m1"); err != nil {
		t.Fatalf("enable failed: %s", err)
	}
	if err := a.Init(); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	if !replaced {
		t.Fatal("override should replace the module implementation")
	}
}

// 验证 Stop 幂等：停机体按注册逆序执行，模块按初始化逆序释放，重复调用安全。
func TestStopIdempotentAndOrder(t *testing.T) {
	var events []string
	a := New("test")
	a.Add(recordModule(&events, "m1", nil))
	a.Add(Module{Name: "m2", Init: func() error {
		events = append(events, "init:m2")
		return nil
	}}) // 无 Release：跳过
	if err := a.Enable("m1", "m2"); err != nil {
		t.Fatalf("enable failed: %s", err)
	}
	a.OnStop(func(time.Duration) { events = append(events, "stop:s1") })
	a.OnStop(func(time.Duration) { events = append(events, "stop:s2") })
	if err := a.Init(); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	a.Stop(time.Second)
	a.Stop(time.Second)
	want := []string{"init:m1", "init:m2", "stop:s2", "stop:s1", "release:m1"}
	if strings.Join(events, ",") != strings.Join(want, ",") {
		t.Fatalf("stop should be idempotent with reverse order: %v", events)
	}
}

// 验证 Init 成功后重复调用幂等（模块不会重复初始化）。
func TestInitIdempotent(t *testing.T) {
	count := 0
	a := New("test")
	a.Add(Module{Name: "m1", Init: func() error { count++; return nil }})
	if err := a.Enable("m1"); err != nil {
		t.Fatalf("enable failed: %s", err)
	}
	if err := a.Init(); err != nil {
		t.Fatalf("init failed: %s", err)
	}
	if err := a.Init(); err != nil {
		t.Fatalf("second init failed: %s", err)
	}
	if count != 1 {
		t.Fatalf("module should be initialized exactly once: %d", count)
	}
}

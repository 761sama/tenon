// Package app 装配层：持有「模块注册表 + 启用清单」与生命周期编排。
// 模块注册制：Add 只注册（不执行），Enable 决定启用哪些、按什么顺序初始化；
// 主服务（如 HTTP）不在模块表内，由预制 app 通过 OnRun/OnStop 固定编排。
package app

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Module 模块定义：名字 + 初始化 + 释放（Release 可为 nil）。
// Init 是闭包，配置在构造模块时捕获。
type Module struct {
	Name    string
	Init    func() error
	Release func()
}

// App 装配单元：模块注册表 + 启用清单 + 运行/停机钩子编排。
type App struct {
	name        string
	mu          sync.Mutex
	modules     map[string]Module     // 模块注册表（名称 -> 模块）
	order       []string              // 模块注册顺序
	enabled     []string              // 启用清单（顺序 = 初始化顺序）
	onRun       []func() error        // 运行体（按注册顺序执行，非阻塞启动）
	onStop      []func(time.Duration) // 停机体（按注册逆序执行）
	inited      []string              // 已成功初始化的模块名（初始化顺序）
	initDone    bool                  // Init 是否已成功（成功后幂等）
	stopTimeout time.Duration         // Run 收到停机信号后执行停机编排的超时时间
	running     atomic.Bool
	stopOnce    sync.Once
}

// 创建装配单元。
// 入参: name (装配单元名称，用于错误信息定位)
// 出参: 装配单元实例
func New(name string) *App {
	return &App{name: name, modules: map[string]Module{}, stopTimeout: 5 * time.Second}
}

// 装配单元名称。
// 出参: 名称
func (a *App) Name() string {
	return a.name
}

// 注册模块（只放进注册表，不执行）；同名重复注册覆盖实现，保留原注册顺序位置。
// 入参: mod (模块定义)
func (a *App) Add(mod Module) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, exists := a.modules[mod.Name]; !exists {
		a.order = append(a.order, mod.Name)
	}
	a.modules[mod.Name] = mod
}

// 模块是否已注册。
// 入参: name (模块名称)
// 出参: 是否已注册
func (a *App) Has(name string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.modules[name]
	return ok
}

// 已注册模块名列表（按注册顺序）。
// 出参: 模块名列表副本
func (a *App) Names() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.order...)
}

// 设置启用清单：顺序 = 初始化顺序；未列出 = 不加载；
// 含未注册名或重复名时返回错误（错误信息列出全部已知模块名）。
// 入参: names (启用模块名清单)
// 出参: 清单非法时返回错误
func (a *App) Enable(names ...string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		if _, dup := seen[n]; dup {
			return fmt.Errorf("app %q: duplicate module %q in enable list; known modules: %s", a.name, n, a.knownLocked())
		}
		seen[n] = struct{}{}
		if _, ok := a.modules[n]; !ok {
			return fmt.Errorf("app %q: unknown module %q; known modules: %s", a.name, n, a.knownLocked())
		}
	}
	a.enabled = append([]string(nil), names...)
	return nil
}

// 当前启用清单。
// 出参: 启用模块名清单副本（未设置时为空）
func (a *App) Enabled() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.enabled...)
}

// 注册运行体（非阻塞启动，如 web.Start），Run 时按注册顺序执行。
// 入参: fn (运行体函数)
func (a *App) OnRun(fn func() error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onRun = append(a.onRun, fn)
}

// 注册停机体，停机时按注册逆序执行。
// 入参: fn (停机体函数，参数为优雅停机超时时间)
func (a *App) OnStop(fn func(time.Duration)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onStop = append(a.onStop, fn)
}

// 设置 Run 收到停机信号后执行停机编排的超时时间（默认 5s）。
// 入参: d (停机超时时间)
func (a *App) SetStopTimeout(d time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if d > 0 {
		a.stopTimeout = d
	}
}

// 按启用清单顺序初始化各模块；任一失败则逆序释放已成功初始化的模块并返回错误。
// 成功后重复调用为幂等空转（直接返回 nil）。
// 出参: 首个失败模块的错误
func (a *App) Init() error {
	a.mu.Lock()
	if a.initDone {
		a.mu.Unlock()
		return nil
	}
	mods := make([]Module, 0, len(a.enabled))
	for _, n := range a.enabled {
		mods = append(mods, a.modules[n])
	}
	a.mu.Unlock()
	var inited []string
	for _, mod := range mods {
		if mod.Init != nil {
			if err := mod.Init(); err != nil {
				a.runReleases(inited)
				return fmt.Errorf("app %q: init module %q failed: %w", a.name, mod.Name, err)
			}
		}
		inited = append(inited, mod.Name)
	}
	a.mu.Lock()
	a.inited = inited
	a.initDone = true
	a.mu.Unlock()
	return nil
}

// 阻塞运行：Init -> 依次执行运行体 -> 等待 SIGINT/SIGTERM -> 停机体逆序执行 -> 模块逆序释放。
// Init 失败时直接返回错误，不启动运行体；运行体启动失败时已初始化的模块按逆序释放。
// 出参: 初始化或运行体启动错误
func (a *App) Run() error {
	if err := a.Init(); err != nil {
		return err
	}
	a.mu.Lock()
	runs := append([]func() error{}, a.onRun...)
	timeout := a.stopTimeout
	a.mu.Unlock()
	for _, fn := range runs {
		if err := fn(); err != nil {
			a.Stop(timeout)
			return err
		}
	}
	a.running.Store(true)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	a.Stop(timeout)
	return nil
}

// 停止装配单元：停机体按注册逆序执行，随后模块按初始化成功的逆序释放（Release 为 nil 的跳过）。
// 幂等：重复调用安全。
// 入参: timeout (优雅停机超时时间，透传给各停机体)
func (a *App) Stop(timeout time.Duration) {
	a.stopOnce.Do(func() {
		a.mu.Lock()
		stops := append([]func(time.Duration){}, a.onStop...)
		inited := append([]string(nil), a.inited...)
		a.mu.Unlock()
		for i := len(stops) - 1; i >= 0; i-- {
			stops[i](timeout)
		}
		a.runReleases(inited)
		a.running.Store(false)
	})
}

// 是否正在运行（Run 进入等待信号阶段后为 true，Stop 后为 false）。
// 出参: 运行状态
func (a *App) IsRunning() bool {
	return a.running.Load()
}

// 按初始化逆序执行各模块的 Release（Release 为 nil 的跳过）。
// 实现为「加锁取快照 -> 释放锁 -> 执行回调」：回调内再操作 App 不会死锁。
// 入参: inited (已成功初始化的模块名，初始化顺序)
func (a *App) runReleases(inited []string) {
	a.mu.Lock()
	var releases []func()
	for i := len(inited) - 1; i >= 0; i-- {
		if r := a.modules[inited[i]].Release; r != nil {
			releases = append(releases, r)
		}
	}
	a.mu.Unlock()
	for _, release := range releases {
		release()
	}
}

// 全部已知模块名（按注册顺序拼接），用于错误信息。
// 出参: 模块名列表字符串（调用方需已持有 a.mu）
func (a *App) knownLocked() string {
	if len(a.order) == 0 {
		return "(none)"
	}
	return strings.Join(a.order, ", ")
}

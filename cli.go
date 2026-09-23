package tenon

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// LaunchFlags 通用启动参数：通过 BindFlags 绑定到 CLI，Execute 解析后生效。
type LaunchFlags struct {
	Debug      bool   // 调试模式（--debug）
	DataDir    string // 数据目录（--data，默认 data）
	ConfigPath string // 配置文件路径（--config，相对路径基于数据目录之外由调用方决定）
}

// CmdFlag 命令行参数定义，参数类型由 Default 的 Go 类型决定。
type CmdFlag struct {
	Name      string // 长选项名
	Shorthand string // 短选项（单字符，可为空）
	Usage     string // 参数说明
	Default   any    // 默认值，支持 string/int/int64/float64/bool/time.Duration
}

// Command 命令定义：支持 flags、位置参数校验与嵌套子命令。
type Command struct {
	Use   string                                  // 用法，如 "unlock <ip>"
	Short string                                  // 简述
	Args  cobra.PositionalArgs                    // 位置参数校验，如 cobra.ExactArgs(1)，nil 表示不校验
	Flags []CmdFlag                               // 命令参数
	Run   func(cmd *cobra.Command, args []string) // 执行函数，flags 值经 cmd.Flags().GetXxx 读取
	Sub   []Command                               // 嵌套子命令
}

// Cli 内置命令行入口：默认提供 server 子命令启动 Web 服务，支持注册自定义子命令。
type Cli struct {
	root *cobra.Command
}

// 创建 CLI 应用。
// 入参: name (应用名), desc (应用描述)
// 出参: CLI 应用实例
func NewCli(name, desc string) *Cli {
	root := &cobra.Command{
		Use:   name,
		Short: desc,
		Long:  desc,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	return &Cli{root: root}
}

// 注册通用启动参数（--debug/--data/--config），返回绑定对象。
// 出参: 启动参数绑定对象（Execute 后值生效）
func (c *Cli) BindFlags() *LaunchFlags {
	f := &LaunchFlags{}
	c.root.PersistentFlags().BoolVar(&f.Debug, "debug", false, "以调试模式启动")
	c.root.PersistentFlags().StringVar(&f.DataDir, "data", "data", "数据目录")
	c.root.PersistentFlags().StringVar(&f.ConfigPath, "config", "", "配置文件路径")
	return f
}

// 获取底层 cobra 根命令，用于框架未封装的能力。
// 出参: cobra 根命令
func (c *Cli) Root() *cobra.Command {
	return c.root
}

// 注册简单子命令（无 flags、无位置参数校验）。
// 入参: use (子命令名), short (简述), run (执行函数)
func (c *Cli) AddCommand(use, short string, run func()) {
	c.Add(Command{
		Use:   use,
		Short: short,
		Run:   func(cmd *cobra.Command, args []string) { run() },
	})
}

// 注册子命令：支持 flags、位置参数校验与嵌套子命令。
// 入参: cmd (命令定义)
// 出参: 构建后的 cobra 命令（可用于进一步定制）
func (c *Cli) Add(cmd Command) *cobra.Command {
	built := buildCommand(cmd)
	c.root.AddCommand(built)
	return built
}

// 按定义构建 cobra 命令（递归处理嵌套子命令）。
// 入参: cmd (命令定义)
// 出参: cobra 命令
func buildCommand(cmd Command) *cobra.Command {
	c := &cobra.Command{
		Use:   cmd.Use,
		Short: cmd.Short,
		Args:  cmd.Args,
	}
	for _, f := range cmd.Flags {
		bindFlag(c, f)
	}
	if cmd.Run != nil {
		c.Run = func(cc *cobra.Command, args []string) { cmd.Run(cc, args) }
	}
	for _, sub := range cmd.Sub {
		c.AddCommand(buildCommand(sub))
	}
	return c
}

// 绑定命令行参数，类型由默认值决定，不支持的类型直接 panic（启动期编程错误应尽早暴露）。
// 入参: c (cobra 命令), f (参数定义)
func bindFlag(c *cobra.Command, f CmdFlag) {
	switch v := f.Default.(type) {
	case string:
		c.Flags().StringP(f.Name, f.Shorthand, v, f.Usage)
	case bool:
		c.Flags().BoolP(f.Name, f.Shorthand, v, f.Usage)
	case int:
		c.Flags().IntP(f.Name, f.Shorthand, v, f.Usage)
	case int64:
		c.Flags().Int64P(f.Name, f.Shorthand, v, f.Usage)
	case float64:
		c.Flags().Float64P(f.Name, f.Shorthand, v, f.Usage)
	case time.Duration:
		c.Flags().DurationP(f.Name, f.Shorthand, v, f.Usage)
	default:
		panic(fmt.Sprintf("tenon: unsupported flag type %T for --%s", f.Default, f.Name))
	}
}

// 注册 server 子命令（启动给定的 Web 服务）并执行 CLI。
// 入参: srv (Web 服务实例)
func (c *Cli) Run(srv *WebServerT) {
	c.root.AddCommand(&cobra.Command{
		Use:   "server",
		Short: "启动 Web 服务",
		Run: func(cmd *cobra.Command, args []string) {
			if err := srv.Run(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	})
	c.Execute()
}

// 执行 CLI（不注册 server 子命令时使用，配合 AddCommand/Add），出错时退出进程。
func (c *Cli) Execute() {
	if err := c.root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// 程序化执行 CLI：不读取 os.Args、出错返回错误而不退出进程（测试与嵌入式场景使用）。
// 入参: args (命令行参数，不含程序名)
// 出参: 执行错误
func (c *Cli) ExecuteArgs(args ...string) error {
	c.root.SetArgs(args)
	return c.root.Execute()
}

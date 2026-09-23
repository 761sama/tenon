package tenon

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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

// 注册自定义子命令。
// 入参: use (子命令名), short (简述), run (执行函数)
func (c *Cli) AddCommand(use, short string, run func()) {
	c.root.AddCommand(&cobra.Command{
		Use:   use,
		Short: short,
		Run:   func(cmd *cobra.Command, args []string) { run() },
	})
}

// LaunchFlags 通用启动参数：通过 BindFlags 绑定到 CLI，Execute 解析后生效。
type LaunchFlags struct {
	Debug      bool   // 调试模式（--debug）
	DataDir    string // 数据目录（--data，默认 data）
	ConfigPath string // 配置文件路径（--config，相对路径基于数据目录之外由调用方决定）
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

// 执行 CLI（不注册 server 子命令时使用，配合 AddCommand）。
func (c *Cli) Execute() {
	if err := c.root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

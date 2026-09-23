package tenon

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// 验证带 flags 与位置参数校验的命令。
func TestCliCommandWithFlags(t *testing.T) {
	cli := NewCli("app", "test")
	var gotIP string
	var gotForce bool
	var gotTimeout time.Duration
	cli.Add(Command{
		Use:   "unlock <ip>",
		Short: "解除 IP 限制",
		Args:  cobra.ExactArgs(1),
		Flags: []CmdFlag{
			{Name: "force", Shorthand: "f", Usage: "强制解除", Default: false},
			{Name: "timeout", Usage: "超时时间", Default: 5 * time.Second},
		},
		Run: func(cmd *cobra.Command, args []string) {
			gotIP = args[0]
			gotForce, _ = cmd.Flags().GetBool("force")
			gotTimeout, _ = cmd.Flags().GetDuration("timeout")
		},
	})
	if err := cli.ExecuteArgs("unlock", "1.2.3.4", "--force", "--timeout", "10s"); err != nil {
		t.Fatalf("execute failed: %s", err)
	}
	if gotIP != "1.2.3.4" || !gotForce || gotTimeout != 10*time.Second {
		t.Fatalf("unexpected values: %q %v %v", gotIP, gotForce, gotTimeout)
	}
	// 缺少位置参数应报错
	if err := cli.ExecuteArgs("unlock"); err == nil {
		t.Fatal("missing positional arg should fail")
	}
	// flags 默认值（cobra 命令实例的 flag 值跨执行保留，需独立实例验证）
	cli2 := NewCli("app", "test")
	var defForce bool
	var defTimeout time.Duration
	cli2.Add(Command{
		Use:   "unlock <ip>",
		Args:  cobra.ExactArgs(1),
		Flags: []CmdFlag{{Name: "force", Default: false}, {Name: "timeout", Default: 5 * time.Second}},
		Run: func(cmd *cobra.Command, args []string) {
			defForce, _ = cmd.Flags().GetBool("force")
			defTimeout, _ = cmd.Flags().GetDuration("timeout")
		},
	})
	if err := cli2.ExecuteArgs("unlock", "5.6.7.8"); err != nil {
		t.Fatalf("execute failed: %s", err)
	}
	if defForce || defTimeout != 5*time.Second {
		t.Fatalf("flag defaults should apply: %v %v", defForce, defTimeout)
	}
}

// 验证嵌套子命令。
func TestCliNestedCommand(t *testing.T) {
	cli := NewCli("app", "test")
	var ran string
	cli.Add(Command{
		Use:   "limit",
		Short: "限制管理",
		Sub: []Command{
			{Use: "list", Short: "列出限制", Run: func(cmd *cobra.Command, args []string) { ran = "list" }},
			{Use: "unlock", Short: "解除限制", Run: func(cmd *cobra.Command, args []string) { ran = "unlock" }},
		},
	})
	if err := cli.ExecuteArgs("limit", "list"); err != nil {
		t.Fatalf("execute failed: %s", err)
	}
	if ran != "list" {
		t.Fatalf("unexpected sub command: %s", ran)
	}
	if err := cli.ExecuteArgs("limit", "unlock"); err != nil {
		t.Fatalf("execute failed: %s", err)
	}
	if ran != "unlock" {
		t.Fatalf("unexpected sub command: %s", ran)
	}
}

// 验证 BindFlags 通用启动参数解析。
func TestCliBindFlags(t *testing.T) {
	cli := NewCli("app", "test")
	flags := cli.BindFlags()
	cli.AddCommand("version", "打印版本", func() {})
	if err := cli.ExecuteArgs("version", "--debug", "--data", "mydata", "--config", "c.json"); err != nil {
		t.Fatalf("execute failed: %s", err)
	}
	if !flags.Debug || flags.DataDir != "mydata" || flags.ConfigPath != "c.json" {
		t.Fatalf("unexpected flags: %+v", flags)
	}
}

// 验证 Root 逃生舱可直接操作 cobra。
func TestCliRoot(t *testing.T) {
	cli := NewCli("app", "test")
	if cli.Root().Use != "app" {
		t.Fatalf("unexpected root: %s", cli.Root().Use)
	}
	ran := false
	cli.Root().AddCommand(&cobra.Command{
		Use: "raw",
		Run: func(cmd *cobra.Command, args []string) { ran = true },
	})
	if err := cli.ExecuteArgs("raw"); err != nil || !ran {
		t.Fatalf("raw cobra command should work: %v, %v", err, ran)
	}
}

// 验证不支持的 flag 类型 panic。
func TestCliUnsupportedFlagType(t *testing.T) {
	cli := NewCli("app", "test")
	defer func() {
		if recover() == nil {
			t.Fatal("unsupported flag type should panic")
		}
	}()
	cli.Add(Command{
		Use:   "bad",
		Flags: []CmdFlag{{Name: "x", Default: []string{}}},
	})
}

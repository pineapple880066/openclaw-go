//go:build ignore
// +build ignore

// Reference copy from goclaw for study and handwritten transcription.
// Source: /Users/pineapple/Desktop/OpenClaw_go/goclaw/cmd/root.go
// Scope: MiniMax provider, Telegram channel, and CLI bootstrap.
// This file is intentionally excluded from the openlaw-go build.

package cmd

// 这份文件是 goclaw CLI 命令树的根。
//
// 它本身不复杂，但它告诉你两件重要的事：
//
// 1. 整个系统的 CLI 入口是怎么组织的
// 2. 用户直接执行 goclaw 时，最后是怎么落到 runGateway() 上的
//
// 阅读时重点看：
//
// - rootCmd 的定义
// - init() 里子命令的挂载顺序
// - resolveConfigPath()
// - Execute()
//
// 你可以把它理解成：
// “命令树的总入口 + 网关启动的最外层壳”
//
import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nextlevelbuilder/goclaw/pkg/protocol"
)

// Version is set at build time via -ldflags "-X github.com/nextlevelbuilder/goclaw/cmd.Version=v1.0.0"
var Version = "dev"

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "goclaw",
	Short: "GoClaw — AI agent gateway",
	Long:  "GoClaw: multi-agent AI platform with WebSocket RPC, tool execution, and channel integration. A Go port of OpenClaw with enhanced security and multi-tenant support.",
	Run: func(cmd *cobra.Command, args []string) {
		runGateway()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: config.json or $GOCLAW_CONFIG)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")

	rootCmd.AddCommand(onboardCmd())
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(pairingCmd())
	rootCmd.AddCommand(agentCmd())
	rootCmd.AddCommand(doctorCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(modelsCmd())
	rootCmd.AddCommand(channelsCmd())
	rootCmd.AddCommand(cronCmd())
	rootCmd.AddCommand(skillsCmd())
	rootCmd.AddCommand(sessionsCmd())
	rootCmd.AddCommand(migrateCmd())
	rootCmd.AddCommand(upgradeCmd())
	rootCmd.AddCommand(authCmd())
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("goclaw %s (protocol %d)\n", Version, protocol.ProtocolVersion)
		},
	}
}

func resolveConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	if v := os.Getenv("GOCLAW_CONFIG"); v != "" {
		return v
	}
	return "config.json"
}

// Execute runs the root cobra command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

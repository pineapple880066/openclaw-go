package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"openclaw-go/pkg/protocol"
)

// Version 先写成 dev
var Version = "dev"

var (
	cfgFile string // 先挂一个配置文件路径 flag，后面 internal/config 会接上它。
	verbose bool  // verbose flag，给 gateway.go 控制日志级别
)

// rootCmd 就是 CLI 的总入口。
// 你可以把它理解成“命令树的根节点”。
// 当用户直接执行 `go run .` 时，没有子命令，就会走这里的 Run。

var rootCmd = &cobra.Command{
	Use: 	"openclaw-go",
	Short: 	"OpenClaw Go rewrite",
	Long:	"OpenClaw Gorewrite: a step-by-step handwritten project inspired by goclaw.",
	Run:	func(cmd *cobra.Command, args []string){
		//业务细节
		runGateway()
	},
}

func init(){
	// PersistentFlags 的意思是：这些 flag 会被根命令和所有子命令共享。
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")

	// 子命令
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(agentCmd())
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use: 	"version",
		Short:	"Print version information",
		Run: func(cmd *cobra.Command, args []string){
			fmt.Printf("openclaw-go %s (protocol %d)\n", Version, protocol.ProtocolVersion)
		},
	}
}

func resolveConfigPath() string {
	// 1. 命令行 --config
	// 2. 环境变量
	// 3. 默认文件名
	if cfgFile != "" {
		return cfgFile
	}
	if v := os.Getenv("OPENCLAW_GO_CONFIG"); v != "" {
		return v
	}
	return "config.json"
}

// Execute() 是main.go的唯一入口

func Execute() {
	if err := rootCmd.Execute(); err != nil {

		// 退出
		os.Exit(1)
	}
}

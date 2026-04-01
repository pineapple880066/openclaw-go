package cmd

import (
	"fmt"
	"os"
	"encoding/json"

	"github.com/spf13/cobra"

	"openclaw-go/internal/config"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: 	"config",
		Short:	"View and manage configuration",
	}
	// 初始化配置
	cmd.AddCommand(configInitCmd())

	// 打印当前生效的配置
	cmd.AddCommand(configShowCmd())

	// 1. path：看程序到底在读哪个配置文件
	// 2. validate：验证当前配置能不能被成功读取
	cmd.AddCommand(configPathCmd())
	cmd.AddCommand(configValidateCmd())

	return cmd
}

// 添加命令
func configPathCmd() *cobra.Command {
	// configPathCmd 用来打印配置文件路径
	return &cobra.Command{
		Use:	"path",
		Short:	"Print the config file path",
		Run:	func(cmd *cobra.Command, args []string) {
			fmt.Println(resolveConfigPath())
		},
	}
}

func configValidateCmd() *cobra.Command {
	// configValidateCmd 用来验证配置文件是否合法。
	return &cobra.Command{
		Use:	"validate",
		Short:	"Validate configuration file",
		Run:	func(cmd *cobra.Command, args []string) {
			cfgPath := resolveConfigPath()

			// 直接调用配置
			// 如果读取失败，还要把错误解析出来
			_, err := config.Load(cfgPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid config: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("Config at %s is valid.\n", cfgPath)
		},
	}
}


func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use: "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string){
			// 先解析配置的路径
			cfgPath := resolveConfigPath()

			// 然后按和主程序完全相同的方式加载配置
			cfg, err := config.Load(cfgPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
				os.Exit(1)
			}

			// 直接格式化为可读的 JSON
			data, err := json.MarshalIndent(cfg, "", " ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding config: %s\n", err)
				os.Exit(1)
			}

			fmt.Println(string(data))
		},
	}
}

func configInitCmd() *cobra.Command {
	return &cobra.Command{
		Use: "init",
		Short: "Create a default config file if it does not exist",
		Run: func(cmd *cobra.Command, args []string){
			cfgPath := resolveConfigPath()

			// 如果文件已经存在，就不需要覆盖
			// Stat 返回 err == nil 则说明存在
			if _, err := os.Stat(cfgPath); err == nil {
				fmt.Fprintf(os.Stderr, "Config already exists at %s\n", cfgPath)
				os.Exit(1)
			} else if !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Error checking config path: %s\n", err)
				os.Exit(1)
			}

			// 直接拿默认配置
			cfg := config.Default()

			// 调用配置层的 Save 写入配置
			if err := config.Save(cfgPath, &cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing config: %s\n", err)
				os.Exit(1)
			}

			// 已经写入 XXX.json 中
			fmt.Printf("Created default config at %s\n", cfgPath)
			
		},
	}
}
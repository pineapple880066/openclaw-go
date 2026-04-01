package cmd

import (
	"fmt"
	"log/slog" // 日志
	"os" // 操作系用接口
	"strings" // 处理日志字符串

	"openclaw-go/internal/config"
)

// runGateway 先只做“假的启动骨架”。

func runGateway(){
	// verbose 来自 root.go 的全局 flag。
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	if lvl := os.Getenv("OPENCLAW_GO_LOG_LEVEL"); lvl != "" {
		switch strings.ToLower(lvl) {
		case "debug":
			logLevel = slog.LevelDebug
		case "info":
			logLevel = slog.LevelInfo
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
	}

	// slog 设置为默认 logger
	// 其他地方可以直接调用 slog.Info(...)
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	// 解析配置路径
	cfgPath := resolveConfigPath()

	// 从配置文件里拿到配置值
	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("filed to load config", "path", cfgPath, "error", err)
		os.Exit(1) // 取失败
	}

	// workspace里的 “～” 去掉,展开
	workspace := cfg.WorkspacePath()

	// 标准化之后的路径写回默认配置
	cfg.Agents.Defaults.Workspace = workspace

	dataDir := cfg.ResolveDataDir()

	cfg.DataDir = dataDir

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		slog.Error("failed to create data dir", "dataDir", dataDir, "error", err)
		os.Exit(1)
	}
	

	if err := os.MkdirAll(workspace, 0755); err != nil {
		slog.Error("failed to create workspace", "workspace", workspace, "error", err)
		os.Exit(1)
	}

	// 生成cfg被读进来的日志，默认值生效
	slog.Info(
		"gateway skeleton starting",
		"config", cfgPath,
		"host", cfg.Gateway.Host,
		"port", cfg.Gateway.Port,
		"workspace", cfg.Agents.Defaults.Workspace,
		"provider", cfg.Agents.Defaults.Provider,
		"model", cfg.Agents.Defaults.Model,
		"maxTurns", cfg.Agents.Defaults.MaxTurns,
		"dataDir", dataDir,
	)


	// 输出测试
	slog.Info("config loaded sucessfully; gateway bootstrap is still TODO")
	fmt.Println()

}
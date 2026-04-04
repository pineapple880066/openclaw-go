package cmd

import (
	"fmt"
	"log/slog" // 日志
	"os" // 操作系用接口
	"strings" // 处理日志字符串
)

// runGateway 先只做“假的启动骨架”。

func runGateway() {
	// 先根据 --verbose 决定默认日志级别。
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	// 再允许环境变量覆盖日志级别。
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

	// 先把默认 logger 装起来。
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))

	// 调统一 helper，把配置和路径一次准备好。
	cfg, paths, err := prepareRuntimeConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// 启动前先确保两个运行目录存在。
	if err := os.MkdirAll(paths.DataDir, 0755); err != nil {
		slog.Error("failed to create data dir", "dataDir", paths.DataDir, "error", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(paths.Workspace, 0755); err != nil {
		slog.Error("failed to create workspace", "workspace", paths.Workspace, "error", err)
		os.Exit(1)
	}

	// 这里是真正把 SQLite store 接进来。
	// 到这一步，gateway 启动时已经会把数据库地基准备好。
	repo, err := openSQLiteRepo(paths)
	if err != nil {
		slog.Error("failed to open sqlite store", "sqlitePath", paths.SQLitePath, "error", err)
		os.Exit(1)
	}
	defer repo.Close()

	// 这条日志现在不只是“配置读到了”，
	// 还表示“运行时目录和 sqlite store 都准备好了”。
	slog.Info(
		"gateway skeleton starting",
		"config", paths.ConfigPath,
		"host", cfg.Gateway.Host,
		"port", cfg.Gateway.Port,
		"workspace", cfg.Agents.Defaults.Workspace,
		"provider", cfg.Agents.Defaults.Provider,
		"model", cfg.Agents.Defaults.Model,
		"maxTurns", cfg.Agents.Defaults.MaxTurns,
		"dataDir", paths.DataDir,
		"sqlitePath", paths.SQLitePath,
	)

	slog.Info("config loaded successfully; sqlite store is ready; gateway bootstrap is still TODO")
	fmt.Println()
}

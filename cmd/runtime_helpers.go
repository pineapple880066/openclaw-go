package cmd

import (
	"path/filepath"

	"openclaw-go/internal/config"
	openlawstore "openclaw-go/internal/store"
)

type runtimePaths struct {
	ConfigPath string
	Workspace  string
	DataDir	   string
	SQLitePath string
}

// prepareRuntimeConfig 的职责是：
// 1. 读取配置
// 2. 把 workspace / dataDir 这些路径标准化
// 3. 推导出 sqlite 文件路径
func prepareRuntimeConfig() (config.Config, runtimePaths, error) {
	// 先解析文件路径
	cfgPath := resolveConfigPath()

	// 读去配置
	cfg, err := config.Load(cfgPath) 
	if err != nil {
		return config.Config{}, runtimePaths{}, err
	}

	// 让 config 层给出已经标准化过的路径。
	workspace := cfg.WorkspacePath()
	dataDir := cfg.ResolveDataDir()

	// 把运行时路径写回 cfg。
	// 这样后面任何地方继续读 cfg，拿到的都是处理后的结果。
	cfg.Agents.Defaults.Workspace = workspace
	cfg.DataDir = dataDir

	paths := runtimePaths{
		ConfigPath: cfgPath,
		Workspace:  workspace,
		DataDir:    dataDir,
		SQLitePath: filepath.Join(dataDir, "openlaw.db"),
	}

	return cfg, paths, nil
}

// openSQLiteRepo 的职责很单纯：
// 给定 dataDir，打开我们当前阶段使用的 SQLite store。
func openSQLiteRepo(paths runtimePaths) (*openlawstore.SQLiteStore, error) {
	return openlawstore.OpenSQLite(paths.SQLitePath)
}


package config

import (
	"path/filepath"

)


func ResolvePath(path string) string {
	// 配置本地路径，处理"~", 尽量转化为绝对路径

	// “～” 展开
	resolved := ExpandHome(path)

	if resolved == "" {
		return resolved
	}

	if filepath.IsAbs(resolved) {
		return resolved
	}

	abs, err := filepath.Abs(resolved)
	if err != nil {
		return resolved
	}

	return abs
}
// 把原始配置里的 workspace 路径做 ExpandHome()。
// 绝对路径转换还暂时留在 cmd/gateway.go，下一步再继续收。

func (c Config) WorkspacePath() string {
	// 展开路径
	return ResolvePath(c.Agents.Defaults.Workspace)
}


func (c Config) ResolveDataDir() string {
	// 给出数据目录的展开
	return ResolvePath(c.DataDir)
}
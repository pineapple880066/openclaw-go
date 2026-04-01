package config

import (
	"encoding/json" // 解析json
	"errors" // 判断文件是否存在
	"os" // 读文件
	"path/filepath" // 

	"fmt"
)

// Load 的职责很单纯：
// 1. 先拿到一份默认配置
// 2. 如果用户没提供配置文件，就直接返回默认配置
// 3. 如果配置文件存在，就把它覆盖进默认配置

func Load(path string) (Config, error) {
	cfg := Default() // 默认配置

	if path == "" {
		// 即使没有 config.json, 也允许环境变量直接驱动程序
		cfg.applyEnvOverrides()
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) { // 文件不存在
			// 没有配置文件时，仍然允许环境变量覆盖默认值。
			cfg.applyEnvOverrides()
			return cfg, nil
		}

		// 读失败（权限不足，路径是目录，磁盘异常
		return Config{}, fmt.Errorf("read config:%w", err)
	}

	if len(data) == 0 {
		cfg.applyEnvOverrides()

		return cfg, nil
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config:%w", err)
	}

	cfg.applyEnvOverrides()
	return cfg, nil
}

// ExpandHome 的作用是把 "~" 开头的路径展开成用户家目录
// 这一步就是在对齐 goclaw/config_load.go 里的同名概念。
func ExpandHome(path string) string {
	// 标准化路径
	if path == "" {
		return path
	}

	if path[0] != '~' { // 开头不含 ~ 就直接返回
		return path
	}

	home, err := os.UserHomeDir() // 拿home目录
	if err != nil {
		return path
	}

	// path == "~" 直接替换成 home
	if path == "~" {
		return home
	}

	// path == "~/xxx" 这种情况， 把 "~/" 替换成 "/Users/xxx/"
	if len(path) >= 2 && path[:2] == "~/" {
		return filepath.Join(home, path[2:])
	}

	return path
}



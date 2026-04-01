package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Save 是 Load 的“写回侧”
// config 包不只负责读配置，也负责把配置安全地写回磁盘。

func Save(path string, cfg * Config) error {
	
	// 格式化为带缩进的 json
	data, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return err
	}

	// 确保目标目录存在。
	// 比如以后 path 变成 ~/.openlaw-go/config.json，这里就会先把目录建好
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 用 0600 写文件

	return os.WriteFile(path, data, 0600)
}
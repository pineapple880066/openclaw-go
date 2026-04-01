package config

import (
	"os"
	"strconv" // port，数字类配置从 string 转 int
)

func (c *Config) applyEnvOverrides() {

	envStr := func(key string, dst *string) {
		if v := os.Getenv(key); v != "" {
			*dst = v
		}
	}

	// 路径配置
	envStr("OPENCLAW_GO_DATA_DIR", &c.DataDir)
	envStr("OPENCLAW_GO_WORKSPACE", &c.Agents.Defaults.Workspace)

	// 模型相关配置
	envStr("OPENCLAW_GO_PROVIDER", &c.Agents.Defaults.Provider)
	envStr("OPENCLAW_GO_MODEL", &c.Agents.Defaults.Model)

	// 网关监听地址
	envStr("OPENCLAW_GO_HOST", &c.Gateway.Host)

	// 端口是 int 
	if v := os.Getenv("OPENCLAW_GO_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil && port > 0 {
			c.Gateway.Port = port
		}
	}

	// max_turns 也是 int
	if v := os.Getenv("OPENCLAW_GO_MAX_TURNS"); v != "" {
		if turns, err := strconv.Atoi(v); err == nil && turns > 0 {
			c.Agents.Defaults.MaxTurns = turns
		}
	}
}
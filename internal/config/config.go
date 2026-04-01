package config

// 配置数据结构
// 把json，yaml， env读进这个结构

// config 结构体
type Config struct {
	DataDir string `json:"data_dir"` // 数据目录 ， sqllite/ cache / 快照
	Agents  AgentsConfig  `json:"agents"`
	Gateway GatewayConfig `json:"gateway"`
}


// defaults 默认配置项
type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
}

// - Workspace：agent 默认工作目录
// - Provider：未来默认 LLM 提供商
// - Model：未来默认模型
// - MaxTurns：先给自己留一个“回合 / 迭代上限”的入口
type AgentDefaults struct {
	Workspace string `json:"workspace"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	MaxTurns  int `json:"max_turns"`
}

// GatewayConfig 表示网关本身的监听配置。
// - Host：监听地址
// - Port：监听端口
type GatewayConfig struct {
	Host string `json:"host"`
	Port int	`json:"port"`
}

// Default 返回一份“程序即使没写 config.json 也能启动”的默认配置。
func Default() Config {
	return Config{
		DataDir: "~/.openclaw/data",
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace:".", // 简单粗暴

				// 模型提供商
				Provider: "openai",
				Model:	  "gpt-40-mini",

				MaxTurns: 8, // 最大调用次数
			},
		},
		Gateway: GatewayConfig{
			Host: "127.0.0.1", // 本地监听
			Port: 18080,
		},
	}
}

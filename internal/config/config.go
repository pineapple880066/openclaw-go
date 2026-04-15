package config


// 你可以把它理解成：程序启动时所有全局配置最终都会收敛到这里。
type Config struct {
	// Config 是当前项目的根配置
	DataDir string `json:"data_dir"`
	// 数据目录 sqlite
	Agents AgentsConfig `json:"agents"`
	Gateway GatewayConfig `json:"gateway"`
	Providers ProvidersConfig `json:"providers"`
}


type AgentsConfig struct {
	// AgentsConfig 目前先只保留 defaults
	Defaults AgentDefaults `json:"defaults"`
}

// 默认配置
type AgentDefaults struct {
	Workspace string `json:"workspace"`
	// 默认工作目录

	// Provider / Model 决定用哪个 LLM
	Provider string `json:"provider"`
	Model    string `json:"model"`

	// MaxTurns 作为最小单轮执行执行次数上限
	MaxTurns int `json:"max_turns"`
}

type GatewayConfig struct {
	// 程序的监听地址
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ProvidersConfig struct {
	OpenAI  ProviderConfig `json:"openai"`
	MiniMax ProviderConfig `json:"minimax"`
}

type ProviderConfig struct {
	// 鉴权， 基础URL， 聊天路径接口
	APIKey 		string `json:"api_key"`
	APIBase 	string `json:"api_base"`
	ChatPath	string `json:"chat_path"`
}


func Default() Config {
	// Default 返回一份“没写 config.json 也能启动”的默认配置
	return Config{
		DataDir: "~/.openclaw/data",

		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace: ".", // 当前项目阶段先用当前目录，最容易理解。
				Provider:  "openai",
				Model:     "gpt-4o-mini",
				MaxTurns:  8,
			},
		},

		Gateway: GatewayConfig{
			Host: "127.0.0.1",
			Port: 18080,
		},

		Providers: ProvidersConfig{
			OpenAI: ProviderConfig{
				// OpenAI 兼容接口默认基址。
				APIBase: "https://api.openai.com/v1",

				// OpenAI 标准聊天路径。
				ChatPath: "/chat/completions",
			},
			MiniMax: ProviderConfig{
				// 这里优先给你国内站的默认地址。
				// 如果你以后用国际站，再通过 env 覆盖掉。
				APIBase: "https://api.minimaxi.com/v1",

				// 这个路径是参考 goclaw 对 MiniMax 的接法。
				ChatPath: "/text/chatcompletion_v2",
			},
		},
	}
}


package providers

import (
	"fmt"
	"strings"

	"openclaw-go/internal/config"
)

// FromConfig 的职责是：
// 根据配置里的 provider ，创建真正的 provider 实例

func FromConfig(providerName string, cfg config.Config) (Provider, error) {
	name := strings.TrimSpace(strings.ToLower(providerName))
	if name == "" {
		name = strings.TrimSpace(strings.ToLower(cfg.Agents.Defaults.Provider))
	}

	switch name {
		case "openai":
			return NewOpenAICompatProvider(
				"openai",
				cfg.Providers.OpenAI.APIKey,
				cfg.Providers.OpenAI.APIBase,
				cfg.Providers.OpenAI.ChatPath,
				cfg.Agents.Defaults.Model,
			), nil

		case "minimax":
			return NewOpenAICompatProvider(
				"minimax",
				cfg.Providers.MiniMax.APIKey,
				cfg.Providers.MiniMax.APIBase,
				cfg.Providers.MiniMax.ChatPath,
				cfg.Agents.Defaults.Model,
			), nil

		default:
			return nil, fmt.Errorf("providers: unsupported provider %q", providerName)
	}
}
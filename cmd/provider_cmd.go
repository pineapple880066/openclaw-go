package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	openlawproviders "openclaw-go/internal/providers"
)

// providerCmd 是 provider 相关命令树。
// 当前先只做一个 verify，专门用来验证 API key / base URL / model 能不能通。

func providerCmd() *cobra.Command {
	cmd := &cobra.Command {
		Use: 	"provider",
		Short:  "Manage and verify providers",
	}

	cmd.AddCommand(providerVerifyCmd())

	return cmd
}

func providerVerifyCmd() *cobra.Command {
	var (
		name 	string
		model	string
		prompt	string
	)

	cmd := &cobra.Command {
		Use:	"verify",
		Short:	"Verify a provider by sending a small chat request",
		Run: func(cmd *cobra.Command, args []string) {
			// 加载配置
			cfg, _, err := prepareRuntimeConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
				os.Exit(1)
			}

			// 如果没显式传 provider / model ，就退回到默认值
			if name == "" {
				name = cfg.Agents.Defaults.Provider
			}
			if model == "" {
				model = cfg.Agents.Defaults.Model
			}
			if prompt == "" {
				prompt = "Reply with exactly: OK"
			}

			// 根据配置创建 provider
			provider, err := openlawproviders.FromConfig(name, cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating provider: %s\n", err)
				os.Exit(1)
			}

			// 给请求加一个超时，避免 provider 一直卡住。
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			resp, err := provider.Chat(ctx, openlawproviders.ChatRequest{
				Model: model,
				Messages: []openlawproviders.Message{
					{
						Role:    "user",
						Content: prompt,
					},
				},
			})

			if err != nil {
				fmt.Fprintf(os.Stderr, "Provider verify failed: %s\n", err)
				os.Exit(1)
			}

			fmt.Printf("Provider verification succeeded:\n")
			fmt.Printf("  Provider: %s\n", resp.Provider)
			fmt.Printf("  Model:    %s\n", resp.Model)
			fmt.Printf("  Reply:    %s\n", resp.Content)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "provider name (default to config)")
	cmd.Flags().StringVar(&model, "model", "", "model name (defaults to config)")
	cmd.Flags().StringVar(&prompt, "prompt", "", "verification prompt")

	return cmd
}
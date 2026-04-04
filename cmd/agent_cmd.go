package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	openlawstore "openclaw-go/internal/store"
)

func agentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Manage agents",
	}

	cmd.AddCommand(agentAddCmd())
	cmd.AddCommand(agentListCmd())

	return cmd
}

// agentAddCmd 用来往 SQLite 里插入一个 agent。
// 这会是你后面 agent loop 的最小入口数据。
func agentAddCmd() *cobra.Command {
	var (
		name         string
		systemPrompt string
		provider     string
		model        string
		workspace    string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new agent",
		Run: func(cmd *cobra.Command, args []string) {
			// 先准备配置和路径
			cfg, paths, err := prepareRuntimeConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
				os.Exit(1)
			}

			// 再打开 SQLite store
			repo, err := openSQLiteRepo(paths)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening sqlite store: %s\n", err)
				os.Exit(1)
			}
			defer repo.Close()

			// 如果用户没显式传 provider / model / workspace，
			// 就退回到 config 里的默认值
			if provider == "" {
				provider = cfg.Agents.Defaults.Provider
			}
			if model == "" {
				model = cfg.Agents.Defaults.Model
			}
			if workspace == "" {
				workspace = cfg.Agents.Defaults.Workspace
			}

			// 组装要写入数据库的 agent 实体。
			agent := openlawstore.Agent{
				Name:         name,
				SystemPrompt: systemPrompt,
				Provider:     provider,
				Model:        model,
				Workspace:    workspace,
			}

			created, err := repo.CreateAgent(context.Background(), agent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating agent: %s\n", err)
				os.Exit(1)
			}

			// 先打印最关键的几个字段。
			// 后面如果你要做 JSON 输出，再扩展。
			fmt.Printf("Created agent:\n")
			fmt.Printf("  ID:        %s\n", created.ID)
			fmt.Printf("  Name:      %s\n", created.Name)
			fmt.Printf("  Provider:  %s\n", created.Provider)
			fmt.Printf("  Model:     %s\n", created.Model)
			fmt.Printf("  Workspace: %s\n", created.Workspace)
		},
	}

	// name 先设成必填。
	// 因为当前阶段一个 agent 至少要有可识别名称。
	cmd.Flags().StringVar(&name, "name", "", "agent name")
	cmd.Flags().StringVar(&systemPrompt, "system-prompt", "", "system prompt")
	cmd.Flags().StringVar(&provider, "provider", "", "provider name (defaults to config)")
	cmd.Flags().StringVar(&model, "model", "", "model name (defaults to config)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "workspace path (defaults to config)")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}



func agentListCmd() *cobra.Command {
	// agentListCmd 用来把数据库里的 agent 全列出来
	// 验证 sqlite 是否真的在工作
	return &cobra.Command{
		Use:   "list",
		Short: "List agents",
		Run: func(cmd *cobra.Command, args []string) {
			_, paths, err := prepareRuntimeConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
				os.Exit(1)
			}

			repo, err := openSQLiteRepo(paths)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening sqlite store: %s\n", err)
				os.Exit(1)
			}
			defer repo.Close()

			agents, err := repo.ListAgents(context.Background())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing agents: %s\n", err)
				os.Exit(1)
			}

			if len(agents) == 0 {
				fmt.Println("No agents found.")
				return
			}

			for _, agent := range agents {
				fmt.Printf("%s\t%s\t%s\t%s\n", agent.ID, agent.Name, agent.Provider, agent.Model)
			}
		},
	}
}
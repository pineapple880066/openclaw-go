package cmd

import (
	"context"
	"fmt"
	openlawstore "openclaw-go/internal/store"
	"os"

	"github.com/spf13/cobra"
)

// 1. add  : 往某条 session 里追加一条消息
// 2. list : 列出某条 session 的全部消息

func messageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "message",
		Short: "Manage session messages",
	}

	cmd.AddCommand(messageAddCmd())
	cmd.AddCommand(messageListCmd())

	return cmd
}

func messageAddCmd() *cobra.Command {
	var (
		sessionID string
		role	  string
		content   string
	)

	cmd := &cobra.Command{
		Use:	"add",
		Short:	"Append a message to a session",
		Run: func(cmd *cobra.Command, args []string) {
			// 准备配置路径
			_, paths, err := prepareRuntimeConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
				os.Exit(1)
			}

			// 打开SQLite store
			repo, err := openSQLiteRepo(paths)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening sqlite store: %s\n", err)
				os.Exit(1)
			}
			defer repo.Close()

			// 确认sessionID确实存在
			_, err = repo.GetSession(context.Background(), sessionID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding session: %s\n", err)
				os.Exit(1)
			}

			message := openlawstore.Message{
				SessionID: sessionID,
				Role:      role,
				Content:   content,
			}

			created, err := repo.AppendMessage(context.Background(), message)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating message: %s\n", err)
				os.Exit(1)
			}

			fmt.Printf("Created message:\n")
			fmt.Printf("  ID:        %s\n", created.ID)
			fmt.Printf("  SessionID: %s\n", created.SessionID)
			fmt.Printf("  Role:      %s\n", created.Role)
			fmt.Printf("  Content:   %s\n", created.Content)
		},
	}
	cmd.Flags().StringVar(&sessionID, "session-id", "", "session ID")
	cmd.Flags().StringVar(&role, "role", "user", "message role")
	cmd.Flags().StringVar(&content, "content", "", "message content")

	_ = cmd.MarkFlagRequired("session-id")
	_ = cmd.MarkFlagRequired("content")

	return cmd
}


func messageListCmd() *cobra.Command {
	// messageListCmd 用来列出某条 session 下面的所有消息。
	// 这是后面做上下文组装、agent loop、Telegram 回复时的基础读取能力。
	var sessionID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages in a session",
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

			// 先确认 session 存在。
			_, err = repo.GetSession(context.Background(), sessionID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding session: %s\n", err)
				os.Exit(1)
			}

			messages, err := repo.ListMessages(context.Background(), sessionID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing messages: %s\n", err)
				os.Exit(1)
			}

			if len(messages) == 0 {
				fmt.Println("No messages found.")
				return
			}

			// 先用最简单的一行一条方式输出。
			// 后面如果你要改成 JSON 或更漂亮的 TUI，再继续扩展。
			for _, message := range messages {
				fmt.Printf("%s\t%s\t%s\n", message.ID, message.Role, message.Content)
			}
		},
	}

	cmd.Flags().StringVar(&sessionID, "session-id", "", "session ID")
	_ = cmd.MarkFlagRequired("session-id")

	return cmd
}
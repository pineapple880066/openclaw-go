package cmd
import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	openlawstore "openclaw-go/internal/store"
)

// 1. create : 创建一条 session
// 2. list   : 列出某个 agent 的所有 session
// 3. show   : 查看某条 session 详情

func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:	"session",
		Short:  "Manage sessions",
	}

	cmd.AddCommand(sessionCreateCmd())
	cmd.AddCommand(sessionListCmd())
	cmd.AddCommand(sessionShowCmd())

	return cmd
}

func sessionCreateCmd() *cobra.Command {
	// sessionCreateCmd 用来给agent新建一个会话: agent_id, channel, peer_id
	var (
		agentID string
		channel string
		peerID	string
		title	string
	)

	cmd := &cobra.Command{
		Use: "create",
		Short: "Create a new session",
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

			// 先检查 agent 是否存在。
			// 这一步不是“多余校验”，而是为了避免插一条指向不存在 agent 的 session。
			_, err = repo.GetAgent(context.Background(), agentID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error finding agent: %s\n", err)
				os.Exit(1)
			}
			// 组装 session 实体。
			session := openlawstore.Session{
				AgentID: agentID,
				Channel: channel,
				PeerID:  peerID,
				Title:   title,
			}

			created, err := repo.CreateSession(context.Background(), session)
			// 新建session： created
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating session: %s\n", err)
				os.Exit(1)
			}

			fmt.Printf("Created session:\n")
			fmt.Printf("  ID:      %s\n", created.ID)
			fmt.Printf("  AgentID: %s\n", created.AgentID)
			fmt.Printf("  Channel: %s\n", created.Channel)
			fmt.Printf("  PeerID:  %s\n", created.PeerID)
			fmt.Printf("  Title:   %s\n", created.Title)
		},
	}

	// agent-id 必填，因为 session 必须属于某个 agent。
	cmd.Flags().StringVar(&agentID, "agent-id", "", "agent ID")
	cmd.Flags().StringVar(&channel, "channel", "cli", "session channel")
	cmd.Flags().StringVar(&peerID, "peer-id", "", "peer ID")
	cmd.Flags().StringVar(&title, "title", "", "session title")

	_ = cmd.MarkFlagRequired("agent-id")
	_ = cmd.MarkFlagRequired("peer-id")

	return cmd
}

func sessionListCmd() *cobra.Command {
	// sessionListCmd 用来按 agent 列出它下面所有 session
	var agentID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sessions for an agent",
		Run: func(cmd *cobra.Command, args []string) {
			_, paths, err := prepareRuntimeConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
				os.Exit(1)
			}

			repo, err := openSQLiteRepo(paths)
			// 找到当前的数据库repo
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening sqlite store: %s\n", err)
				os.Exit(1)
			}
			defer repo.Close()

			sessions, err := repo.ListSessionsByAgent(context.Background(), agentID)
			// 调用数据库查找
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error listing sessions: %s\n", err)
				os.Exit(1)
			}

			if len(sessions) == 0 {
				// 没有session
				fmt.Println("No sessions found.")
				return
			}

			for _, session := range sessions {
				// 输出所有找到的 sessions
				fmt.Printf(
					"%s\t%s\t%s\t%s\n",
					session.ID,
					session.Channel,
					session.PeerID,
					session.Title,
				)
			}
		},
	}

	cmd.Flags().StringVar(&agentID, "agent-id", "", "agent ID")
	_ = cmd.MarkFlagRequired("agent-id") // 起码需要agent_id

	return cmd
}

func sessionShowCmd() *cobra.Command {
	var sessionID string

	cmd := &cobra.Command {
		Use:	"show",
		Short:	"Show session details",
		Run: func(cmd *cobra.Command, args []string){
			_, paths, err := prepareRuntimeConfig()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
				os.Exit(1)
			}

			repo, err := openSQLiteRepo (paths)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error opening sqlite store: %s\n", err)
				os.Exit(1)
			}
			defer repo.Close()

			session, err := repo.GetSession(context.Background(), sessionID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting session: %s\n", err)
				os.Exit(1)
			}

			fmt.Printf("Session:\n")
			fmt.Printf("  ID:        %s\n", session.ID)
			fmt.Printf("  AgentID:   %s\n", session.AgentID)
			fmt.Printf("  Channel:   %s\n", session.Channel)
			fmt.Printf("  PeerID:    %s\n", session.PeerID)
			fmt.Printf("  Title:     %s\n", session.Title)
			fmt.Printf("  CreatedAt: %s\n", session.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  UpdatedAt: %s\n", session.UpdatedAt.Format("2006-01-02 15:04:05"))
		},
	}

	cmd.Flags().StringVar(&sessionID, "id", "", "session ID")
	_ = cmd.MarkFlagRequired("id")

	return cmd
}
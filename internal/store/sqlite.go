package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)


type SQLiteStore struct {
	// 单机持久化
	db *sql.DB
}

func OpenSQLite(path string) (*SQLiteStore, error) {
	// 打开 SQLite 文件并且自动初始化 schema

	if path == "" {
		// 路径不存在
		return nil, errors.New("store: empty")
	}

	// 先确定数据库文件目录存在
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	store := &SQLiteStore{db: db}

	if err := store.init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil

}

func (s *SQLiteStore) Close() error {
	// 关闭db
	return s.db.Close()
}

func (s *SQLiteStore) init(ctx context.Context) error {
	// SQL 建表以及索引
	statements := []string {
		`PRAGMA foreign_keys = ON;`,
		// 这句打开后，删除 agent 时，相关 session/message 才能按外键规则联动。

		// agent表,保存基础配置
		`CREATE TABLE IF NOT EXISTS agents(
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			system_prompt TEXT NOT NULL DEFAULT '',
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			workspace TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		// session表， 保存某个 agent 聊天的会话
		`CREATE TABLE IF NOT EXISTS sessions(
			id TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL,
			channel TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY(agent_id) REFERENCES agents(id) ON DELETE CASCADE

		);`,
		// 索引， 按agent_id 查session 会很频繁，所以先建索引
		`CREATE INDEX IF NOT EXISTS idx_sessions_agent_id
		ON sessions(agent_id);`,

		// messages 表：保存会话里的每一条消息
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY(session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);`,

		// 按 session_id + created_at 排序取消息历史是标准操作，先建联合索引
		`CREATE INDEX IF NOT EXISTS idx_messages_session_id_created_at 
		ON messages(session_id, created_at);`,

	}

	for _,stmt := range statements {
		// 每条 SQL 单独执行
		// 这样出错时更容易定位是哪张表或哪个索引有问题
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("init sqlite schema: %w", err)
		}
	}

	return nil
}

func (s *SQLiteStore) CreateAgent(ctx context.Context, agent Agent) (Agent, error) {
	// 先做最小字段校验
	// 这些字段在当前阶段缺了就没法形成一个可运行 agent
	if agent.Name == "" {
		return Agent{}, errors.New("store: agent name is required")
	}
	if agent.Provider == "" {
		return Agent{}, errors.New("store: agent provider is required")
	}
	if agent.Model == "" {
		return Agent{}, errors.New("store: agent model is required")
	}
	if agent.Workspace == "" {
		return Agent{}, errors.New("store: agent workspace is required")
	}

	// 统一用 UTC，避免以后 Telegram / HTTP / CLI 混在一起时出现时区比较问题。
	now := time.Now().UTC()
	if agent.ID == "" {
		agent.ID = uuid.NewString() // 生成唯一标识符
	}

	// 如果外面没传 ID，就在 store 层生成。
	// 这样调用方可以简单一点。
	if agent.ID == "" {
		agent.ID = uuid.NewString()
	}

	// Create 时 created_at 只在空值时补一次。
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	agent.UpdatedAt = now

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO agents(id, name, system_prompt, provider, model, workspace,created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		agent.ID,
		agent.Name,
		agent.SystemPrompt,
		agent.Provider,
		agent.Model,
		agent.Workspace,
		agent.CreatedAt.Format(time.RFC3339Nano),
		agent.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Agent{}, fmt.Errorf("insert agent: %w", err)
	}

	return agent, nil
}

func (s *SQLiteStore) GetAgent(ctx context.Context, id string) (Agent, error) {
	// QueryRowContext 用来查询单条记录
	// 如果查不到，会在scan的时候返回 sql.ErrNoRows
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, name, system_prompt, provider, model, workspace, created_at, updated_at
		FROM agents
		WHERE id = ?`,
		id,
	)

	agent, err := scanAgent(row)
	if err != nil {
		// 统一把“数据库没有这条数据”转换成 ErrNotFound
		if errors.Is(err, sql.ErrNoRows) {
			return Agent{}, ErrNotFound
		}
		return Agent{}, fmt.Errorf("get agent: %w", err)
	}

	return agent, nil
}

func (s *SQLiteStore) ListAgents(ctx context.Context) ([]Agent, error) {
	// 列表查询，用 QueryContext
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, name, system_prompt, provider, model, workspace, created_at, updated_at
		FROM agents
		ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan agent row: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

func (s *SQLiteStore) CreateSession(ctx context.Context, session Session) (Session, error) {
	// session 必须至少知道属于哪个agent,来自哪个channel， 对应哪个 peer
	if session.AgentID == "" {
		return Session{}, errors.New("store: session agent_id is required")
	}
	if session.Channel == "" {
		return Session{}, errors.New("store: session channel is required")
	}
	if session.PeerID == "" {
		return Session{}, errors.New("store: session peer_id is required")
	}

	now := time.Now().UTC()

	if session.ID == "" {
		session.ID = uuid.NewString()
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, agent_id, channel, peer_id, title, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		session.ID,
		session.AgentID,
		session.Channel,
		session.PeerID,
		session.Title,
		session.CreatedAt.Format(time.RFC3339Nano),
		session.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Session{}, fmt.Errorf("insert session: %w", err)
	}

	return session, nil
}

func (s *SQLiteStore) GetSession(ctx context.Context, id string) (Session, error) {
	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, agent_id, peer_id, title, created_at, updated_at
		FROM sessions
		WHERE id = ?`,
		id,
	)

	session, err := scanSession(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrNotFound
		}
		return Session{}, fmt.Errorf("get session: %w", err)
	}
	return session, nil
}

func (s *SQLiteStore) ListSessionsByAgent(ctx context.Context, agentID string) ([]Session, error) {
	// 返回 agentID 对应的 所有sessions
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, agent_id, channel, peer_id, title, created_at, updated_at
		 FROM sessions
		 WHERE agent_id = ?
		 ORDER BY created_at ASC`,
		agentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan session row: %w", err)
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return sessions, nil
}

func (s *SQLiteStore) AppendMessage(ctx context.Context, message Message) (Message, error) {
	// 插入一个新 session

	// 一条消息至少要知道属于哪个 session、是什么角色、内容是什么。
	if message.SessionID == "" {
		return Message{}, errors.New("store: message session_id is required")
	}
	if message.Role == "" {
		return Message{}, errors.New("store: message role is required")
	}
	if message.Content == "" {
		return Message{}, errors.New("store: message content is required")
	}

	if message.ID == "" {
		message.ID = uuid.NewString()
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = time.Now().UTC()
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO messages (id, session_id, role, content, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		message.ID,
		message.SessionID,
		message.Role,
		message.Content,
		message.CreatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Message{}, fmt.Errorf("insert message: %w", err)
	}

	return message, nil
}

func (s *SQLiteStore) ListMessages(ctx context.Context, sessionID string) ([]Message, error) {
	// 通过 session_id 查 messages 对话记录
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, session_id, role, content, created_at
		 FROM messages
		 WHERE session_id = ?
		 ORDER BY created_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		message, err := scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan message row: %w", err)
		}
		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, nil
}


type scanner interface {
	// scanner 实现 scanAgent / scanSession / scanMessage / QueryRow
	Scan(dest ...any) error
}

func scanAgent(s scanner) (Agent, error) {
	var agent Agent
	var createdAt string
	var updatedAt string

	if err := s.Scan(
		&agent.ID,
		&agent.Name,
		&agent.SystemPrompt,
		&agent.Provider,
		&agent.Model,
		&agent.Workspace,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Agent{}, err
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Agent{}, fmt.Errorf("parse agent created_at: %w", err)
	}
	parsedUpdatedAt, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Agent{}, fmt.Errorf("parse agent uodated_at: %w", err)
	}

	agent.CreatedAt = parsedCreatedAt
	agent.UpdatedAt = parsedUpdatedAt
	return agent, nil
}

func scanSession(s scanner) (Session, error) {
	var session Session
	var createdAt string
	var updatedAt string
	
	if err := s.Scan(
		&session.ID,
		&session.AgentID,
		&session.Channel,
		&session.PeerID,
		&session.Title,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Session{}, err
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse session created_at: %w", err)
	}
	parsedUreatedAt, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Session{}, fmt.Errorf("parse session updated_at: %w", err)
	}

	session.CreatedAt = parsedCreatedAt
	session.UpdatedAt = parsedUreatedAt
	return session, nil
}

func scanMessage(s scanner) (Message, error) {
	var message Message
	var createdAt string

	if err := s.Scan(
		&message.ID,
		&message.SessionID,
		&message.Role,
		&message.Content,
		&createdAt,
	); err != nil {
		return Message{}, err
	}

	parsedCreatedAt, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Message{}, fmt.Errorf("parse message created_at: %w", err)
	}

	message.CreatedAt = parsedCreatedAt
	return message, nil
}


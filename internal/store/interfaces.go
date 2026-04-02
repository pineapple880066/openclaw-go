package store

import (
	"context"
	"errors"
)

// ErrNotFound 表示“数据库里面没有这个记录”
var ErrNotFound = errors.New("store: not found")

// Repository 是这个 store 的总接口: agent / session / message
type Repository interface {
	AgentRepository
	SessionRepository
	MessageRepository
	Close() error
}

type AgentRepository interface {
	CreateAgent(ctx context.Context, agent Agent) (Agent, error)
	GetAgent(ctx context.Context, id string) (Agent, error)
	ListAgents(ctx context.Context) ([]Agent, error)

}
type SessionRepository interface {
	CreateSession(ctx context.Context, session Session) (Session, error)
	GetSession(ctx context.Context, id string) (Session, error)
	ListSessionByAgent(ctx context.Context, agentID string) ([]Session, error)

}

type MessageRepository interface {
	AppendMessage(ctx context.Context, message Message) (Message, error)
	ListMessages(ctx context.Context, sessionID string) ([]Message, error)
}
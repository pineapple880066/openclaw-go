package store

import(
	"time"
)


type Agent struct {
	// 最基础的实体：身份，模型，工作区，系统提示词
	ID				string		`json:"id"`
	Name			string		`json:"name"`
	SystemPrompt	string		`json:"system_prompt"`
	Provider		string		`json:"provider"`
	Model			string		`json:"model"`
	Workspace		string		`json:"workspace"`
	CreatedAt		time.Time	`json:"created_at"`
	UpdatedAt		time.Time	`json:"updated_at"`
}



type Session struct {
	// session 表示某个agent和某个对话对象之间的一条对话
	ID				string		`json:"id"`
	AgentID			string		`json:"agent_id"`
	Channel			string		`json:"channel"`
	PeerID			string		`json:"peer_id"`
	Title			string		`json:"title"`
	CreatedAt		time.Time	`json:"created_at"`
	UpdatedAt		time.Time	`json:"updated_at"`
}



type Message struct {
	// message 是会话session里的单条消息
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

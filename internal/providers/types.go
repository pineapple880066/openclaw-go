package providers

import (
	"context"
)

type Message struct {
	// 发给模型的一条消息
	Role	string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	// ChatRequest 是一次聊天请求
	Model 		string
	Messages	[]Message
	Temperature float64
}

type ChatResponse struct {
	// ChatResponse 是模型返回的最小结构。
	Provider string
	Model 	 string
	Content	 string
}

// Provider 是 provider 的抽象接口
type Provider interface {
	Name() string
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
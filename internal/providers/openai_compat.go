package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAICompatProvider struct {
	name		string
	apiKey  	string
	apiBase      string
	chatPath     string
	defaultModel string
	client       *http.Client
}

func NewOpenAICompatProvider(
	// 创建一个 OepnAI-compatible provider
	name string,
	apiKey string,
	apiBase string,
	chatPath string,
	defaultModel string,
) *OpenAICompatProvider {
	if chatPath == "" {
		chatPath = "/chat/completions"
	}

	return &OpenAICompatProvider{
		name:         name,
		apiKey:       apiKey,
		apiBase:      strings.TrimRight(apiBase, "/"),
		chatPath:     chatPath,
		defaultModel: defaultModel,
		client: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (p *OpenAICompatProvider) Name() string {
	// 获得名字 Name
	return p.name
}


type openAICompatChatRequest struct {
	// openAICompatChatRequest 是发给兼容接口的请求体
	Model		string    `json:"model"`
	Messages		[]Message `json:"message"`
	Temperature *float64  `json:"temperature,omitempty"`
}

type openAICompatChatResponse struct {
	// 解析的响应结构
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		}	`json:"message"`
	}	`json:"choices"`
}

func (p *OpenAICompatProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// 发起聊天请求
	if p.apiKey == "" {
		return ChatResponse{}, errors.New("provider: missing API key")
	}

	// 没传 model， 退回默认
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}
	if model == "" {
		return ChatResponse{}, errors.New("providers: missing model")
	}

	// 当前阶段至少要有一条 message。
	if len(req.Messages) == 0 {
		return ChatResponse{}, errors.New("providers: empty messages")
	}

	payload := openAICompatChatRequest{
		Model:		model,
		Messages: req.Messages,
	}

	if req.Temperature != 0 {
		payload.Temperature = &req.Temperature
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal chat request: %w", err)
	}

	// 最终请求地址
	endpoint := p.apiBase + p.chatPath

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("build chat request: %w", err)
	}

	httpReq.Header.Set("Authorzation", "Bearer " + p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("do chat request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read chat response: %w", err)
	}

	// 非 2xx 直接把响应体带出来，方便调试 provider 错误
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{}, fmt.Errorf(
			"provider http %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(respBody)),
		)
	}

	var parsed openAICompatChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return ChatResponse{}, fmt.Errorf("parse chat reponse: %w", err)
	}

	if len(parsed.Choices) == 0 {
		return ChatResponse{}, errors.New("providers: empty choices")
	}

	content := strings.TrimSpace(parsed.Choices[0].Message.Content)

	return ChatResponse{
		Provider: p.name,
		Model:	model,
		Content: content,
	}, nil

}
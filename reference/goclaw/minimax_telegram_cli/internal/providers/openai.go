//go:build ignore
// +build ignore

// Reference copy from goclaw for study and handwritten transcription.
// Source: /Users/pineapple/Desktop/OpenClaw_go/goclaw/internal/providers/openai.go
// Scope: MiniMax provider, Telegram channel, and CLI bootstrap.
// This file is intentionally excluded from the openlaw-go build.

package providers

// 这份文件是 MiniMax 那条链里最关键的一份。
//
// 你要先建立一个正确认知：
//
// 1. goclaw 没有单独写一个 minimax.go
// 2. MiniMax 走的是这个 OpenAI-compatible provider
// 3. 真正让它“变成 MiniMax”的，不是 provider 类型不同
// 4. 而是注册时给了不同的 base URL、default model、chatPath
//
// 所以如果你后面想在 openlaw-go 里接 MiniMax，
// 真正要抄的是这份文件的思路和结构，
// 然后再看 cmd/gateway_providers.go 里它是怎么被注册出来的。
//
// 阅读顺序建议：
//
// 1. 先看 OpenAIProvider 结构体本身
// 2. 再看 NewOpenAIProvider
// 3. 再看 WithChatPath
// 4. 再看 Chat / ChatStream
// 5. 最后看 buildRequestBody / doRequest / parseResponse
//
// 这几个点连起来，你就会明白：
// “OpenAI-compatible provider” 在 goclaw 里到底承担了什么。
//
import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// OpenAIProvider implements Provider for OpenAI-compatible APIs
// (OpenAI, Groq, OpenRouter, DeepSeek, VLLM, etc.)
type OpenAIProvider struct {
	name         string
	apiKey       string
	apiBase      string
	chatPath     string // defaults to "/chat/completions"
	defaultModel string
	providerType string // DB provider_type (e.g. "gemini_native", "openai", "minimax_native")
	client       *http.Client
	retryConfig  RetryConfig
}

// NewOpenAIProvider 是“构造通用 OpenAI-compatible provider 实例”的入口。
//
// 你可以把它理解成：
//
// - name：这个 provider 在 registry 里的名字
// - apiKey：鉴权凭据
// - apiBase：基础 URL
// - defaultModel：默认模型名
//
// 注意这里还没有出现 MiniMax 专属逻辑。
// MiniMax 的差异不是在这里写死的，而是后面靠：
//
// - WithChatPath(...)
// - 注册时传入的 base URL / default model
//
// 也正因为这样，goclaw 才能用一套 OpenAI-compatible provider 复用很多厂商。
func NewOpenAIProvider(name, apiKey, apiBase, defaultModel string) *OpenAIProvider {
	if apiBase == "" {
		apiBase = "https://api.openai.com/v1"
	}
	apiBase = strings.TrimRight(apiBase, "/")

	return &OpenAIProvider{
		name:         name,
		apiKey:       apiKey,
		apiBase:      apiBase,
		chatPath:     "/chat/completions",
		defaultModel: defaultModel,
		client:       &http.Client{Timeout: DefaultHTTPTimeout},
		retryConfig:  DefaultRetryConfig(),
	}
}

// WithChatPath returns a copy with a custom chat completions path (e.g. "/text/chatcompletion_v2" for MiniMax native API).
//
// 这就是 MiniMax 真正和普通 OpenAI 路径拉开差异的位置之一。
// 也就是说：
//
// - OpenAI 默认走 /chat/completions
// - MiniMax 在注册时把路径切到 /text/chatcompletion_v2
//
// 所以后面你读 gateway_providers.go 时，一定要把那一行和这个函数连起来看。
func (p *OpenAIProvider) WithChatPath(path string) *OpenAIProvider {
	p.chatPath = path
	return p
}

func (p *OpenAIProvider) Name() string           { return p.name }
func (p *OpenAIProvider) DefaultModel() string   { return p.defaultModel }
func (p *OpenAIProvider) SupportsThinking() bool { return true }
func (p *OpenAIProvider) APIKey() string         { return p.apiKey }
func (p *OpenAIProvider) APIBase() string        { return p.apiBase }
func (p *OpenAIProvider) ProviderType() string   { return p.providerType }

// WithProviderType sets the DB provider_type for correct API endpoint routing in media tools.
func (p *OpenAIProvider) WithProviderType(pt string) *OpenAIProvider {
	p.providerType = pt
	return p
}

// resolveModel returns the model ID to use for a request.
// For OpenRouter, model IDs require a provider prefix (e.g. "anthropic/claude-sonnet-4-5-20250929").
// If the caller passes an unprefixed model, fall back to the provider's default.
func (p *OpenAIProvider) resolveModel(model string) string {
	if model == "" {
		return p.defaultModel
	}
	if p.name == "openrouter" && !strings.Contains(model, "/") {
		return p.defaultModel
	}
	return model
}

// Chat 是“非流式请求”的总入口。
//
// 它内部真正做的事是 4 步：
//
// 1. 先决定最终要用哪个 model
// 2. 再把内部 ChatRequest 组装成 provider 的 HTTP 请求体
// 3. 调用带重试的 request 执行逻辑
// 4. 最后把 provider 的响应解析成内部 ChatResponse
//
// 所以你如果想快速理解 provider 主链，
// 先看 Chat，然后顺着跳去：
//
// - buildRequestBody
// - doRequest
// - parseResponse
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	model := p.resolveModel(req.Model)
	body := p.buildRequestBody(model, req, false)

	chatFn := p.chatRequestFn(ctx, body)

	resp, err := RetryDo(ctx, p.retryConfig, chatFn)

	// Auto-clamp max_tokens and retry once if the model rejects the value
	if err != nil {
		if clamped := clampMaxTokensFromError(err, body); clamped {
			slog.Info("max_tokens clamped, retrying", "model", model, "limit", clampedLimit(body))
			return RetryDo(ctx, p.retryConfig, chatFn)
		}
	}

	return resp, err
}

// chatRequestFn returns a closure that performs a single non-streaming chat request.
// Shared between initial attempt and post-clamp retry to avoid duplication.
//
// 这层 closure 的意义是：
//
// RetryDo 需要一个“可以被重复调用”的函数。
// 所以 goclaw 把一次真实请求封装成闭包，这样：
//
// - 第一次失败可以重试
// - max_tokens 被自动收缩后还能再跑一次
func (p *OpenAIProvider) chatRequestFn(ctx context.Context, body map[string]any) func() (*ChatResponse, error) {
	return func() (*ChatResponse, error) {
		respBody, err := p.doRequest(ctx, body)
		if err != nil {
			return nil, err
		}
		defer respBody.Close()

		var oaiResp openAIResponse
		if err := json.NewDecoder(respBody).Decode(&oaiResp); err != nil {
			return nil, fmt.Errorf("%s: decode response: %w", p.name, err)
		}

		return p.parseResponse(&oaiResp), nil
	}
}

// ChatStream 是“流式请求”的总入口。
//
// 它和 Chat 的差别不是只有 stream=true 这么简单，
// 还包括：
//
// - SSE 的读取
// - chunk 的拼接
// - tool call 的累计
// - thinking / reasoning 内容的累计
// - usage 的最终收集
//
// 所以如果你只想先弄懂“最小请求是怎么发出去的”，
// 可以先只读 Chat，不急着先吞下 ChatStream。
func (p *OpenAIProvider) ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	model := p.resolveModel(req.Model)
	body := p.buildRequestBody(model, req, true)

	// Retry only the connection phase; once streaming starts, no retry.
	respBody, err := RetryDo(ctx, p.retryConfig, func() (io.ReadCloser, error) {
		return p.doRequest(ctx, body)
	})

	// Auto-clamp max_tokens and retry once if the model rejects the value
	if err != nil {
		if clamped := clampMaxTokensFromError(err, body); clamped {
			slog.Info("max_tokens clamped, retrying stream", "model", model, "limit", clampedLimit(body))
			respBody, err = RetryDo(ctx, p.retryConfig, func() (io.ReadCloser, error) {
				return p.doRequest(ctx, body)
			})
		}
	}
	if err != nil {
		return nil, err
	}
	defer respBody.Close()

	result := &ChatResponse{FinishReason: "stop"}
	accumulators := make(map[int]*toolCallAccumulator)

	scanner := bufio.NewScanner(respBody)
	scanner.Buffer(make([]byte, 0, SSEScanBufInit), SSEScanBufMax)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		// SSE spec allows both "data: value" and "data:value" (space is optional).
		// Some providers (e.g. Kimi) omit the space after the colon.
		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimPrefix(data, " ")
		if data == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// Usage chunk often has empty choices — extract usage before skipping.
		// When stream_options.include_usage is true, the final chunk contains
		// usage data but choices is typically an empty array.
		if chunk.Usage != nil {
			result.Usage = &Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
			if chunk.Usage.PromptTokensDetails != nil {
				result.Usage.CacheReadTokens = chunk.Usage.PromptTokensDetails.CachedTokens
			}
			if chunk.Usage.CompletionTokensDetails != nil && chunk.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
				result.Usage.ThinkingTokens = chunk.Usage.CompletionTokensDetails.ReasoningTokens
			}
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta.ReasoningContent != "" {
			result.Thinking += delta.ReasoningContent
			if onChunk != nil {
				onChunk(StreamChunk{Thinking: delta.ReasoningContent})
			}
		}
		if delta.Content != "" {
			result.Content += delta.Content
			if onChunk != nil {
				onChunk(StreamChunk{Content: delta.Content})
			}
		}

		// Accumulate streamed tool calls
		for _, tc := range delta.ToolCalls {
			acc, ok := accumulators[tc.Index]
			if !ok {
				acc = &toolCallAccumulator{
					ToolCall: ToolCall{ID: tc.ID, Name: strings.TrimSpace(tc.Function.Name)},
				}
				accumulators[tc.Index] = acc
			}
			if tc.Function.Name != "" {
				acc.Name = strings.TrimSpace(tc.Function.Name)
			}
			acc.rawArgs += tc.Function.Arguments
			if tc.Function.ThoughtSignature != "" {
				acc.thoughtSig = tc.Function.ThoughtSignature
			}
		}

		if chunk.Choices[0].FinishReason != "" {
			result.FinishReason = chunk.Choices[0].FinishReason
		}

	}

	// Check for scanner errors (timeout, connection reset, etc.)
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s: stream read error: %w", p.name, err)
	}

	// Parse accumulated tool call arguments
	for i := 0; i < len(accumulators); i++ {
		acc := accumulators[i]
		args := make(map[string]any)
		if err := json.Unmarshal([]byte(acc.rawArgs), &args); err != nil && acc.rawArgs != "" {
			slog.Warn("openai_stream: failed to parse tool call arguments",
				"tool", acc.Name, "raw_len", len(acc.rawArgs), "error", err)
		}
		acc.Arguments = args
		if acc.thoughtSig != "" {
			acc.Metadata = map[string]string{"thought_signature": acc.thoughtSig}
		}
		result.ToolCalls = append(result.ToolCalls, acc.ToolCall)
	}

	if len(result.ToolCalls) > 0 {
		result.FinishReason = "tool_calls"
	}

	if onChunk != nil {
		onChunk(StreamChunk{Done: true})
	}

	return result, nil
}

// buildRequestBody 是整个 provider 最值得精读的函数之一。
//
// 因为内部 `ChatRequest` 和外部 OpenAI-compatible HTTP JSON
// 不是一一对应的，必须在这里做“协议翻译”。
//
// 它负责的事包括：
//
// - 把内部 Message 转成 OpenAI wire format
// - 处理 tool_calls
// - 处理 image 输入
// - 处理 reasoning / thinking 相关字段
// - 处理 stream_options
// - 把统一 Options 翻译成不同 provider 需要的字段
//
// 你要真正看懂 provider，是绕不开这一个函数的。
func (p *OpenAIProvider) buildRequestBody(model string, req ChatRequest, stream bool) map[string]any {
	// Gemini 2.5+: collapse tool_call cycles missing thought_signature.
	// Gemini requires thought_signature echoed back on every tool_call; models that
	// don't return it (e.g. gemini-3-flash) will cause HTTP 400 if sent as-is.
	// Tool results are folded into plain user messages to preserve context.
	inputMessages := req.Messages

	// Compute provider capability once: does this endpoint support Google's thought_signature?
	// We check providerType, name, apiBase, and the model string (robust detection for proxies/OpenRouter).
	supportsThoughtSignature := strings.Contains(strings.ToLower(p.providerType), "gemini") ||
		strings.Contains(strings.ToLower(p.name), "gemini") ||
		strings.Contains(strings.ToLower(p.apiBase), "generativelanguage") ||
		strings.Contains(strings.ToLower(model), "gemini")

	if supportsThoughtSignature {
		inputMessages = collapseToolCallsWithoutSig(inputMessages)
	}

	// Convert messages to proper OpenAI wire format.
	// This is necessary because our internal Message/ToolCall structs don't match
	// the OpenAI API format (tool_calls need type+function wrapper, arguments as JSON string).
	// Also omits empty content on assistant messages with tool_calls (Gemini compatibility).
	msgs := make([]map[string]any, 0, len(inputMessages))
	for _, m := range inputMessages {
		msg := map[string]any{
			"role": m.Role,
		}

		// Echo reasoning_content for assistant messages (required by Kimi, DeepSeek when thinking is enabled)
		if m.Thinking != "" && m.Role == "assistant" {
			msg["reasoning_content"] = m.Thinking
		}

		// Include content; omit empty content for assistant messages with tool_calls
		// (Gemini rejects empty content → "must include at least one parts field").
		if m.Role == "user" && len(m.Images) > 0 {
			var parts []map[string]any
			for _, img := range m.Images {
				parts = append(parts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": fmt.Sprintf("data:%s;base64,%s", img.MimeType, img.Data),
					},
				})
			}
			if m.Content != "" {
				parts = append(parts, map[string]any{
					"type": "text",
					"text": m.Content,
				})
			}
			msg["content"] = parts
		} else if m.Content != "" || len(m.ToolCalls) == 0 {
			msg["content"] = m.Content
		}

		// Convert tool_calls to OpenAI wire format:
		// {id, type: "function", function: {name, arguments: "<json string>"}}
		if len(m.ToolCalls) > 0 {
			toolCalls := make([]map[string]any, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				argsJSON, _ := json.Marshal(tc.Arguments)
				fn := map[string]any{
					"name":      tc.Name,
					"arguments": string(argsJSON),
				}
				if sig := tc.Metadata["thought_signature"]; sig != "" {
					// Only send thought_signature to providers that support it (Google/Gemini).
					// Non-Google providers will reject the unknown field with 422 Unprocessable Entity.
					if supportsThoughtSignature {
						fn["thought_signature"] = sig
					}
				}
				toolCalls[i] = map[string]any{
					"id":       tc.ID,
					"type":     "function",
					"function": fn,
				}
			}
			msg["tool_calls"] = toolCalls
		}

		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}

		msgs = append(msgs, msg)
	}

	// Safety net: strip trailing assistant message to prevent HTTP 400 from
	// proxy providers (LiteLLM, OpenRouter) that don't support assistant prefill.
	// This should rarely trigger — the agent loop ensures user message is last.
	if len(msgs) > 0 {
		if role, _ := msgs[len(msgs)-1]["role"].(string); role == "assistant" {
			slog.Warn("openai: stripped trailing assistant message (unsupported prefill)",
				"provider", p.name, "model", model)
			msgs = msgs[:len(msgs)-1]
		}
	}

	body := map[string]any{
		"model":    model,
		"messages": msgs,
		"stream":   stream,
	}

	if len(req.Tools) > 0 {
		body["tools"] = CleanToolSchemas(p.name, req.Tools)
		body["tool_choice"] = "auto"
	}

	if stream {
		body["stream_options"] = map[string]any{
			"include_usage": true,
		}
	}

	// Merge options
	if v, ok := req.Options[OptMaxTokens]; ok {
		if strings.HasPrefix(model, "gpt-5") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4") {
			body["max_completion_tokens"] = v
		} else {
			body["max_tokens"] = v
		}
	}
	if v, ok := req.Options[OptTemperature]; ok {
		// GPT-5 mini/nano and o-series models only support default temperature
		skipTemp := strings.HasPrefix(model, "gpt-5-mini") || strings.HasPrefix(model, "gpt-5-nano") || strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4")
		if !skipTemp {
			body["temperature"] = v
		}
	}

	// Inject reasoning_effort for o-series models (ignored by models that don't support it)
	if level, ok := req.Options[OptThinkingLevel].(string); ok && level != "" && level != "off" {
		body[OptReasoningEffort] = level
	}

	// DashScope-specific passthrough keys
	if v, ok := req.Options[OptEnableThinking]; ok {
		body[OptEnableThinking] = v
	}
	if v, ok := req.Options[OptThinkingBudget]; ok {
		body[OptThinkingBudget] = v
	}

	return body
}

// doRequest 负责真正把 HTTP 请求发出去。
//
// 这层只做传输层相关的事：
//
// - JSON 编码
// - 组 URL
// - 设置请求头
// - 发请求
// - 处理非 200 错误
//
// 它不负责解释响应语义。
// 响应语义留给 parseResponse。
func (p *OpenAIProvider) doRequest(ctx context.Context, body any) (io.ReadCloser, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", p.name, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+p.chatPath, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%s: create request: %w", p.name, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	// Azure OpenAI/Foundry support for now atleast
	if strings.Contains(strings.ToLower(p.apiBase), "azure.com") {
		httpReq.Header.Set("api-key", p.apiKey)
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s: request failed: %w", p.name, err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		retryAfter := ParseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, &HTTPError{
			Status:     resp.StatusCode,
			Body:       fmt.Sprintf("%s: %s", p.name, string(respBody)),
			RetryAfter: retryAfter,
		}
	}

	return resp.Body, nil
}

// parseResponse 负责把 OpenAI-compatible 响应结构翻译回内部 ChatResponse。
//
// 注意它做的不是“简单读 content”而已，还包括：
//
// - reasoning_content -> Thinking
// - tool_calls -> []ToolCall
// - usage -> Usage
// - finish_reason 的规范化
//
// 所以 provider 的“返回值长什么样”，真正是在这里统一下来的。
func (p *OpenAIProvider) parseResponse(resp *openAIResponse) *ChatResponse {
	result := &ChatResponse{FinishReason: "stop"}

	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		result.Content = msg.Content
		result.Thinking = msg.ReasoningContent
		result.FinishReason = resp.Choices[0].FinishReason

		for _, tc := range msg.ToolCalls {
			args := make(map[string]any)
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil && tc.Function.Arguments != "" {
				slog.Warn("openai: failed to parse tool call arguments",
					"tool", tc.Function.Name, "raw_len", len(tc.Function.Arguments), "error", err)
			}
			call := ToolCall{
				ID:        tc.ID,
				Name:      strings.TrimSpace(tc.Function.Name),
				Arguments: args,
			}
			if tc.Function.ThoughtSignature != "" {
				call.Metadata = map[string]string{"thought_signature": tc.Function.ThoughtSignature}
			}
			result.ToolCalls = append(result.ToolCalls, call)
		}

		if len(result.ToolCalls) > 0 {
			result.FinishReason = "tool_calls"
		}
	}

	if resp.Usage != nil {
		result.Usage = &Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
		if resp.Usage.PromptTokensDetails != nil {
			result.Usage.CacheReadTokens = resp.Usage.PromptTokensDetails.CachedTokens
		}
		if resp.Usage.CompletionTokensDetails != nil && resp.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
			result.Usage.ThinkingTokens = resp.Usage.CompletionTokensDetails.ReasoningTokens
		}
	}

	return result
}

// maxTokensLimitRe matches "supports at most N completion tokens" from OpenAI 400 errors.
var maxTokensLimitRe = regexp.MustCompile(`supports at most (\d+) completion tokens`)

// clampMaxTokensFromError checks if an error is a 400 "max_tokens is too large" rejection.
// If so, it parses the model's stated limit, clamps the body's max_tokens/max_completion_tokens,
// and returns true so the caller can retry.
func clampMaxTokensFromError(err error, body map[string]any) bool {
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.Status != http.StatusBadRequest {
		return false
	}
	if !strings.Contains(httpErr.Body, "max_tokens") || !strings.Contains(httpErr.Body, "too large") {
		return false
	}

	matches := maxTokensLimitRe.FindStringSubmatch(httpErr.Body)
	if len(matches) < 2 {
		return false
	}
	limit, parseErr := strconv.Atoi(matches[1])
	if parseErr != nil || limit <= 0 {
		return false
	}

	// Clamp whichever key is present
	if _, ok := body["max_completion_tokens"]; ok {
		body["max_completion_tokens"] = limit
	} else {
		body["max_tokens"] = limit
	}
	return true
}

// clampedLimit returns the clamped max_tokens or max_completion_tokens value for logging.
func clampedLimit(body map[string]any) any {
	if v, ok := body["max_completion_tokens"]; ok {
		return v
	}
	return body["max_tokens"]
}

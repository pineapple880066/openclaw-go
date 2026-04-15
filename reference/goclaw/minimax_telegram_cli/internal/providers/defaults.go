//go:build ignore
// +build ignore

// Reference copy from goclaw for study and handwritten transcription.
// Source: /Users/pineapple/Desktop/OpenClaw_go/goclaw/internal/providers/defaults.go
// Scope: MiniMax provider, Telegram channel, and CLI bootstrap.
// This file is intentionally excluded from the openlaw-go build.

package providers

import "time"

// Provider-level defaults for HTTP clients and stream parsing.
const (
	// DefaultHTTPTimeout is the HTTP client timeout for LLM API calls.
	// 5 minutes allows for long-running streaming responses with extended thinking.
	DefaultHTTPTimeout = 300 * time.Second

	// SSE stream scanner buffer sizes (OpenAI-compat, Anthropic, Codex).
	SSEScanBufInit = 64 * 1024    // 64KB initial buffer
	SSEScanBufMax  = 1024 * 1024  // 1MB max line for large tool call / thinking chunks

	// Stdio/JSONRPC scanner buffer sizes (Claude CLI, ACP).
	StdioScanBufInit = 256 * 1024       // 256KB initial buffer
	StdioScanBufMax  = 10 * 1024 * 1024 // 10MB max for large protocol messages
)

//go:build ignore
// +build ignore

// Reference copy from goclaw for study and handwritten transcription.
// Source: /Users/pineapple/Desktop/OpenClaw_go/goclaw/internal/channels/telegram/stt.go
// Scope: MiniMax provider, Telegram channel, and CLI bootstrap.
// This file is intentionally excluded from the openlaw-go build.

package telegram

import (
	"context"

	"github.com/nextlevelbuilder/goclaw/internal/channels/media"
)

// transcribeAudio calls the configured STT proxy service with the given audio file and returns
// the transcribed text. Returns ("", nil) silently when STT is not configured or filePath is empty.
// Delegates to the shared media.TranscribeAudio implementation.
func (c *Channel) transcribeAudio(ctx context.Context, filePath string) (string, error) {
	return media.TranscribeAudio(ctx, media.STTConfig{
		ProxyURL:       c.config.STTProxyURL,
		APIKey:         c.config.STTAPIKey,
		TenantID:       c.config.STTTenantID,
		TimeoutSeconds: c.config.STTTimeoutSeconds,
	}, filePath)
}

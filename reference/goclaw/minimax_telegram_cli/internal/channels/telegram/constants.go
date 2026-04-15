//go:build ignore
// +build ignore

// Reference copy from goclaw for study and handwritten transcription.
// Source: /Users/pineapple/Desktop/OpenClaw_go/goclaw/internal/channels/telegram/constants.go
// Scope: MiniMax provider, Telegram channel, and CLI bootstrap.
// This file is intentionally excluded from the openlaw-go build.

package telegram

import "time"

const (
	// telegramMaxMessageLen is the safe limit for Telegram messages.
	// Telegram's hard limit is 4096, but we use 4000 for safety (matching TS textChunkLimit).
	telegramMaxMessageLen = 4000

	// telegramCaptionMaxLen is the max length for media captions.
	telegramCaptionMaxLen = 1024

	// pairingReplyDebounce is the minimum interval between pairing replies to the same user.
	pairingReplyDebounce = 60 * time.Second

	// sendOverallTimeout is the maximum duration for a multi-retry send sequence.
	sendOverallTimeout = 60 * time.Second

	// probeOverallTimeout is the maximum duration for initial bot status check and command sync.
	probeOverallTimeout = 60 * time.Second
)

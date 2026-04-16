//go:build ignore
// +build ignore

// Reference copy from goclaw for study and handwritten transcription.
// Source: /Users/pineapple/Desktop/OpenClaw_go/goclaw/internal/channels/telegram/stream.go
// Scope: MiniMax provider, Telegram channel, and CLI bootstrap.
// This file is intentionally excluded from the openlaw-go build.

package telegram

// 这份文件是 Telegram 流式输出链。
//
// 它处理的是：
// agent 正在生成内容时，Telegram 端怎么逐步看到输出。
//
// 对 Telegram 这种平台来说，流式输出不是简单地“打印 chunk”，
// 还要处理：
//
// - 发送草稿还是发送普通消息
// - 连续 edit 的频率控制
// - group / DM 场景差异
// - 最终结果怎么接管占位消息
//
// 所以这份文件的意义在于：
// 它把“LLM stream chunk”翻译成“Telegram 用户能看见的实时反馈”。
//
import (
	"context"
	"log/slog"
	"math"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/nextlevelbuilder/goclaw/internal/channels"
)

const (
	// defaultStreamThrottle is the minimum delay between message edits (matching TS: 1000ms).
	defaultStreamThrottle = 1000 * time.Millisecond

	// streamMaxChars is the max message length for streaming (Telegram limit).
	streamMaxChars = 4096

	// draftIDMax is the maximum value for draft_id before wrapping.
	draftIDMax = math.MaxInt32
)

// nextDraftID is a global atomic counter for sendMessageDraft draft_id values.
// Each streaming session gets a unique ID (matching TS pattern: 1 → Int32 max, wraps).
var nextDraftID atomic.Int32

// allocateDraftID returns a unique draft_id for sendMessageDraft.
func allocateDraftID() int {
	for {
		cur := nextDraftID.Load()
		next := cur + 1
		if next >= int32(draftIDMax) {
			next = 1
		}
		if nextDraftID.CompareAndSwap(cur, next) {
			return int(next)
		}
	}
}

// draftFallbackRe matches Telegram API errors indicating sendMessageDraft is unsupported.
// Ref: TS src/telegram/draft-stream.ts fallback patterns.
var draftFallbackRe = regexp.MustCompile(`(?i)(unknown method|method.*not (found|available|supported)|unsupported|can't be used|can be used only)`)

// shouldFallbackFromDraft returns true if the error indicates sendMessageDraft
// is permanently unavailable and the stream should fall back to message transport.
func shouldFallbackFromDraft(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	if !strings.Contains(msg, "sendmessagedraft") && !strings.Contains(msg, "send_message_draft") {
		return false
	}
	return draftFallbackRe.MatchString(err.Error())
}

// DraftStream manages a streaming preview message that gets edited as content arrives.
// Ref: TS src/telegram/draft-stream.ts → createTelegramDraftStream()
//
// Supports two transports:
//   - Draft transport (sendMessageDraft): Preferred for DMs. Ephemeral preview, no real message created.
//   - Message transport (sendMessage + editMessageText): Fallback. Creates a real message that can be edited.
//
// State machine:
//
//	NOT_STARTED → first Update() → sendMessageDraft or sendMessage → STREAMING
//	STREAMING   → subsequent Update() → sendMessageDraft or editMessageText (throttled) → STREAMING
//	STREAMING   → Stop() → final flush → STOPPED
//	STREAMING   → Clear() → deleteMessage (message transport only) → DELETED
type DraftStream struct {
	bot             *telego.Bot
	chatID          int64
	messageThreadID int           // forum topic thread ID (0 = no thread)
	messageID       int           // 0 = not yet created (message transport only)
	lastText        string        // last sent text (for dedup)
	throttle        time.Duration // min delay between edits
	lastEdit        time.Time
	mu              sync.Mutex
	stopped         bool
	pending         string // pending text to send (buffered during throttle)
	draftID         int    // sendMessageDraft draft_id (0 = message transport)
	useDraft        bool   // true = draft transport, false = message transport
	draftFailed     bool   // true = draft API rejected permanently, using message transport
	sendMayHaveLanded bool   // true = initial sendMessage was attempted and may have landed (even if timed out)
}

// NewDraftStream creates a new streaming preview manager.
// When useDraft is true, the stream will attempt to use sendMessageDraft (Bot API 9.3+)
// and automatically fall back to sendMessage+editMessageText if the API rejects it.
//
// NewDraftStream 是 Telegram 流式输出对象的构造入口。
//
// 这个对象的意义不是“保存所有最终结果”，
// 而是充当一层中间缓冲：
//
// - LLM chunk 持续进来
// - DraftStream 节流聚合
// - 再以编辑消息 / 草稿的方式逐步刷新 Telegram 端显示
func NewDraftStream(bot *telego.Bot, chatID int64, throttleMs int, messageThreadID int, useDraft bool) *DraftStream {
	throttle := defaultStreamThrottle
	if throttleMs > 0 {
		throttle = time.Duration(throttleMs) * time.Millisecond
	}
	var draftID int
	if useDraft {
		draftID = allocateDraftID()
	}
	return &DraftStream{
		bot:             bot,
		chatID:          chatID,
		messageThreadID: messageThreadID,
		throttle:        throttle,
		useDraft:        useDraft,
		draftID:         draftID,
	}
}

// Update sends or edits the streaming message with the latest text.
// Throttled to avoid hitting Telegram rate limits.
func (ds *DraftStream) Update(ctx context.Context, text string) {
	// Update 只负责“把最新文本塞进缓冲并安排刷新”，
	// 它自己不保证每次都立刻发请求。
	//
	// 这是 Telegram 流式输出里很关键的节流点：
	// 不然模型每吐一个小 chunk 就调一次 API，会非常抖也非常浪费请求。
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if ds.stopped {
		return
	}

	// Truncate to Telegram max
	if len(text) > streamMaxChars {
		text = text[:streamMaxChars]
	}

	// Dedup: skip if text unchanged
	if text == ds.lastText {
		return
	}

	ds.pending = text

	// Check throttle
	if time.Since(ds.lastEdit) < ds.throttle {
		return
	}

	ds.flush(ctx)
}

// Flush forces sending the pending text immediately.
func (ds *DraftStream) Flush(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.flush(ctx)
}

// flush sends/edits the pending text (must hold mu lock).
func (ds *DraftStream) flush(ctx context.Context) error {
	// flush 才是真正把缓冲内容推到 Telegram API 的地方。
	//
	// 也就是说：
	// Update 是“写入最新状态”，
	// flush 是“把最新状态落到 Telegram”。
	if ds.pending == "" || ds.pending == ds.lastText {
		return nil
	}

	text := ds.pending
	htmlText := markdownToTelegramHTML(text)

	// --- Draft transport (sendMessageDraft) ---
	if ds.useDraft && !ds.draftFailed {
		params := &telego.SendMessageDraftParams{
			ChatID:    ds.chatID,
			DraftID:   ds.draftID,
			Text:      htmlText,
			ParseMode: telego.ModeHTML,
		}
		if sendThreadID := resolveThreadIDForSend(ds.messageThreadID); sendThreadID > 0 {
			params.MessageThreadID = sendThreadID
		}
		if err := ds.bot.SendMessageDraft(ctx, params); err != nil {
			if shouldFallbackFromDraft(err) {
				// Permanent fallback to message transport
				slog.Warn("stream: sendMessageDraft unavailable, falling back to message transport", "error", err)
				ds.draftFailed = true
				// Fall through to message transport below
			} else {
				slog.Debug("stream: sendMessageDraft failed", "error", err)
				return err
			}
		} else {
			ds.lastText = text
			ds.lastEdit = time.Now()
			return nil
		}
	}

	// --- Message transport (sendMessage + editMessageText) ---
	if ds.messageID == 0 {
		// First message: send new
		// TS ref: buildTelegramThreadParams() — General topic (1) must be omitted.
		params := &telego.SendMessageParams{
			ChatID:    tu.ID(ds.chatID),
			Text:      htmlText,
			ParseMode: telego.ModeHTML,
		}
		if sendThreadID := resolveThreadIDForSend(ds.messageThreadID); sendThreadID > 0 {
			params.MessageThreadID = sendThreadID
		}
		ds.sendMayHaveLanded = true
		msg, err := ds.bot.SendMessage(ctx, params)
		// TS ref: withTelegramThreadFallback — retry without thread ID when topic is deleted.
		if err != nil && params.MessageThreadID != 0 && threadNotFoundRe.MatchString(err.Error()) {
			slog.Warn("stream: thread not found, retrying without message_thread_id", "thread_id", params.MessageThreadID)
			params.MessageThreadID = 0
			msg, err = ds.bot.SendMessage(ctx, params)
		}
		if err != nil {
			if isPostConnectNetworkErr(err) {
				slog.Warn("stream: initial sendMessage timed out or lost. Treating as landed to avoid duplicate.", "error", err)
				return nil // treat as successful but with unknown messageID
			}
			slog.Debug("stream: failed to send initial message", "error", err)
			return err
		}
		ds.messageID = msg.MessageID
	} else {
		// Edit existing message
		editMsg := tu.EditMessageText(tu.ID(ds.chatID), ds.messageID, htmlText)
		editMsg.ParseMode = telego.ModeHTML
		if _, err := ds.bot.EditMessageText(ctx, editMsg); err != nil {
			// Ignore "not modified" errors
			if !messageNotModifiedRe.MatchString(err.Error()) {
				slog.Debug("stream: failed to edit message", "error", err)
			}
		}
	}

	ds.lastText = text
	ds.lastEdit = time.Now()
	return nil
}

// Stop finalizes the stream with a final edit.
func (ds *DraftStream) Stop(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.stopped = true
	return ds.flush(ctx)
}

// Clear stops the stream and deletes the message (message transport only).
// Draft transport has no persistent message to delete.
func (ds *DraftStream) Clear(ctx context.Context) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.stopped = true
	if ds.messageID != 0 {
		_ = ds.bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
			ChatID:    tu.ID(ds.chatID),
			MessageID: ds.messageID,
		})
		ds.messageID = 0
	}
	return nil
}

// MessageID returns the streaming message ID (0 if not yet created or using draft transport).
func (ds *DraftStream) MessageID() int {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.messageID
}

// UsedDraftTransport returns true if the stream is (or was) using draft transport
// and didn't fall back to message transport.
func (ds *DraftStream) UsedDraftTransport() bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.useDraft && !ds.draftFailed
}

// --- StreamingChannel implementation ---

// CreateStream prepares a per-run streaming handle for the given chatID (localKey).
// Implements channels.StreamingChannel.
//
// For DMs: seeds the stream with the "Thinking..." placeholder messageID so that
// flush() uses editMessageText to update it progressively. This gives a smooth
// transition: "Thinking..." → streaming chunks → (Send() edits final formatted response).
//
// For groups: deletes the placeholder and lets the stream create its own message,
// since group placeholders drift away as other messages arrive.
func (c *Channel) CreateStream(ctx context.Context, chatID string, firstStream bool) (channels.ChannelStream, error) {
	// CreateStream 把 Telegram Channel 接入 channels.ChannelStream 统一抽象。
	//
	// 这意味着上层 agent/gateway 不需要知道 Telegram 是怎么实现流式输出的，
	// 只要知道：
	// “这个 channel 能不能给我一个 stream 对象”。
	//
	// 所以它是“Telegram 具体实现”和“上层统一流接口”之间的桥。
	id, err := parseRawChatID(chatID)
	if err != nil {
		return nil, err
	}

	// Look up thread ID stored during handleMessage
	threadID := 0
	if v, ok := c.threadIDs.Load(chatID); ok {
		threadID = v.(int)
	}

	isDM := id > 0

	// Draft transport only for non-first streams (answer lane) in DMs.
	// First stream must use message transport because it may become the
	// reasoning lane — draft messages are ephemeral and would disappear
	// when the answer stream starts.
	useDraft := isDM && !firstStream && c.draftTransportEnabled()
	ds := NewDraftStream(c.bot, id, 0, threadID, useDraft)

	// No placeholder seeding — DraftStream creates its own message on first flush().
	// This avoids "reply to deleted/non-existent message" artifacts.

	return ds, nil
}

// FinalizeStream hands the stream's messageID back to the placeholders map so that Send()
// can edit it with the properly formatted final response.
// Also stops any thinking animation for the chat.
// Implements channels.StreamingChannel.
func (c *Channel) FinalizeStream(ctx context.Context, chatID string, stream channels.ChannelStream) {
	// FinalizeStream 负责把“流式阶段”平稳收尾到“最终消息状态”。
	//
	// 这一步很重要，因为流式阶段和最终消息阶段通常不是完全同一个对象语义。
	// 如果收尾做不好，就会出现：
	//
	// - 重复发最终消息
	// - 占位消息没有被接管
	// - 最终文本和流式文本不一致
	//
	// 所以这一步本质上是在做“stream lifecycle 的最后一次状态移交”。
	msgID := stream.MessageID()
	if msgID != 0 {
		// Hand off the stream message to Send() for final formatted edit.
		c.placeholders.Store(chatID, msgID)
		slog.Info("stream: ended, handing off to Send()", "chat_id", chatID, "message_id", msgID)
	} else if ds, ok := stream.(*DraftStream); ok && ds.sendMayHaveLanded && !ds.UsedDraftTransport() {
		// The message transport was used but no ID was retrieved (timeout).
		// We MUST store a -1 placeholder to signal to Send() that a message
		// likely landed and it should NOT send a duplicate, even if it cannot edit.
		c.placeholders.Store(chatID, -1)
		slog.Warn("stream: initial send landed but ID unknown. Suppressing fallback message to avoid duplicate.", "chat_id", chatID)
	}

	// Capture draft ID for clearing after the final Send()
	if ds, ok := stream.(*DraftStream); ok && ds.UsedDraftTransport() {
		c.pendingDraftID.Store(chatID, ds.draftID)
	}

	// Stop thinking animation
	if stop, ok := c.stopThinking.Load(chatID); ok {
		if cf, ok := stop.(*thinkingCancel); ok {
			cf.Cancel()
		}
		c.stopThinking.Delete(chatID)
	}
}

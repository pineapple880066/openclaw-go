# goclaw 参考副本：MiniMax + Telegram + CLI

这个目录不是你当前 `openlaw-go` 的运行代码。

它的用途只有一个：

把 `goclaw` 里和你当前目标最相关的 3 块真实源码，单独拎出来放在一个不会参与编译的位置，方便你直接阅读、对照、手抄。

## 为什么放在这里

- 你现在的 `openlaw-go` 还在早期骨架阶段。
- 直接把 `goclaw` 这些文件接进现有编译链，会立刻把大量额外依赖一起拉进来。
- 所以这里先放一份“参考副本”，而不是直接替换你现在的实现。

所有 `.go` 文件顶部都加了：

- `//go:build ignore`
- `// +build ignore`

这表示：

- VS Code 里可以正常看代码
- 但 `go run .`、`go build`、`go mod tidy` 不会把这里当成当前项目的一部分去编译

## 这份参考副本包含什么

### 1. CLI 启动链

- `cmd/root.go`
- `cmd/gateway.go`
- `cmd/gateway_providers.go`

这 3 个文件对应的是：

- CLI 根命令怎么挂起来
- 网关主启动链怎么装配
- provider 是怎么注册进系统的

如果你现在最关心 MiniMax 是怎么接进去的，先看：

- `cmd/gateway_providers.go`

## 2. MiniMax Provider 真实实现路径

MiniMax 在 `goclaw` 里不是单独的 `minimax.go`。

它的真实做法是：

1. 使用 `internal/providers/openai.go` 里的 `OpenAIProvider`
2. 注册时指定：
   - provider name = `minimax`
   - base URL = `https://api.minimax.io/v1`
   - default model = `MiniMax-M2.5`
3. 再通过 `WithChatPath("/text/chatcompletion_v2")` 切到 MiniMax 的聊天接口路径

为了能完整读懂这条链，这里一起保留了：

- `internal/providers/types.go`
- `internal/providers/defaults.go`
- `internal/providers/retry.go`
- `internal/providers/registry.go`
- `internal/providers/openai_types.go`
- `internal/providers/schema_cleaner.go`
- `internal/providers/openai_gemini.go`
- `internal/providers/openai.go`

建议阅读顺序：

1. `internal/providers/types.go`
2. `internal/providers/registry.go`
3. `internal/providers/openai_types.go`
4. `internal/providers/openai.go`
5. `cmd/gateway_providers.go`

## 3. Telegram Channel 真实实现路径

这里保留的是 `goclaw/internal/channels/telegram/` 的非测试文件。

它体现的不是“单个 bot handler”，而是一整套真实生产通道：

- Bot 初始化
- Long polling
- 消息解析
- mention gating
- 分组 / 主题 / thread 路由
- typing / reaction / stream / media / stt
- pairing / writer / task 这些命令入口

建议阅读顺序：

1. `internal/channels/telegram/factory.go`
2. `internal/channels/telegram/channel.go`
3. `internal/channels/telegram/handlers.go`
4. `internal/channels/telegram/send.go`
5. `internal/channels/telegram/stream.go`
6. `internal/channels/telegram/commands.go`

## 你接下来怎么用这份目录

推荐分 2 种用法：

### 用法 A：纯阅读对照

你先在这里把 `goclaw` 的真实结构看顺，再决定你自己的 `openlaw-go` 要不要照搬。

### 用法 B：分模块迁移

后面如果你要我继续动手，我会按这个目录为唯一来源，逐块把真实实现迁到你当前项目，而不是再给你“我自己设计的简化版”。

## 当前最关键的 3 个入口

- MiniMax 注册入口：`cmd/gateway_providers.go`
- OpenAI-compatible provider 核心：`internal/providers/openai.go`
- Telegram 通道入口：`internal/channels/telegram/channel.go`

## 注意

这份目录里的 import 仍然保留了 `goclaw` 原始路径。

原因很简单：

- 这份目录的目标是“保留真实源码结构供你阅读”
- 不是“立刻参与当前项目编译”

等你决定开始真正迁移某一块时，再统一改 import 路径和依赖接线。

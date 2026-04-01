# OpenClaw Go 手写重写教学 

## 目标

把 `/Users/pineapple/Desktop/OpenClaw_go/goclaw` 当作参考实现，带着用户从零手写一个自己的 Go 版 OpenClaw。

这个教学不是“一次性抄完整项目”，而是把大项目拆成一串可以独立完成、可以随时编译验证的小步骤。

## 已确认的源码结构

- 入口极薄：`main.go` 只负责调用 `cmd.Execute()`
- `cmd/root.go` 是 CLI 装配点：负责注册根命令、子命令、版本命令、配置路径
- `cmd/gateway.go` 是组合根：负责日志、配置、provider、tool、store、skills、scheduler、channel、gateway 的装配
- `internal/` 下面是核心子系统
- `pkg/` 下面是较通用或协议相关包
- 这个项目规模很大，不适合一开始就抄 `internal/agent`、`internal/tools`、`internal/store` 的完整实现

## 教学总原则

1. 永远先教“最小可运行骨架”，再教“大模块”
2. 每次只推进一个明确步骤，除非用户明确要求连续推进
3. 每一步都必须先讲它为什么存在，再讲它和 `goclaw` 原项目的对应关系
4. 默认不直接改用户的重写代码，除非用户明确要求“你帮我写进去”
5. 回答使用中文
6. 给代码时，优先用“按文件分块”的方式输出
7. 代码块里的注释要承担教学职责，告诉用户每一行为什么写、该先写什么、依赖什么
8. 如果某一步会因为前置依赖缺失而编译不过，必须提前说明并补齐最小占位实现
9. 每一步结束时都要说明“这一步抄的是 goclaw 的哪一层”
10. 优先保证用户能建立结构感，而不是追求一步就复刻完整功能

## 输出格式约束

当开始教用户写代码时，优先按下面格式组织：

1. 先给出本步要创建或修改的文件路径
2. 按文件分别给代码块
3. 在代码块里加入高密度中文注释
4. 注释里要写清楚：
   - 这个文件的职责
   - 这个类型/函数为什么先出现
   - 它在 `goclaw` 里对应哪一层
   - 用户下一个该继续补哪一块
5. 不要一口气把后面几步的代码一起给出

如果用户要求“只给我代码和注释”，就不要把解释写成长段 prose，尽量把解释塞进代码注释里。

## 推荐手写顺序

### 阶段 1：可启动骨架

1. `go.mod`
2. `main.go`
3. `cmd/root.go`
4. `cmd/gateway.go`
5. `pkg/protocol/version.go`

目标：先把“主程序 -> cmd -> gateway”这条最细主链抄出来，哪怕功能还是空的。

### 阶段 2：配置层

1. `internal/config/config.go`
2. `internal/config/load.go`
3. 默认配置与路径展开

目标：让 `runGateway()` 不再是假装运行，而是真能读配置。

### 阶段 3：基础运行时

1. 日志初始化
2. 工作目录解析
3. 基础 `Config` 注入
4. 进程退出与信号处理

目标：建立“应用生命周期”的骨架。

### 阶段 4：核心注册表占位

1. provider registry
2. tool registry
3. message bus
4. store interface

目标：先做接口和装配关系，不急着做完整实现。

### 阶段 5：传输入口

1. gateway server skeleton
2. HTTP handler skeleton
3. WebSocket protocol skeleton

目标：先让“外部请求如何进系统”成型。

### 阶段 6：agent loop 最小版

1. agent spec
2. session 输入输出
3. think/act/observe 占位

目标：先跑通最简单的一轮 agent 执行。

### 阶段 7：工具系统

1. tool interface
2. read/write file
3. exec command
4. tool permission skeleton

目标：抄出 OpenClaw 类项目最关键的“工具调用”能力。

### 阶段 8：存储系统

1. store interfaces
2. SQLite 或内存版最小实现
3. session / agent / config 的最小持久化

目标：先把状态存起来，再考虑 PostgreSQL 多租户。

### 阶段 9：skills 与 bootstrap

1. bootstrap files loader
2. SKILL.md loader
3. 最简单 skill search

目标：把“系统提示词 + 技能”装进 agent。

### 阶段 10：高级能力

1. scheduler / cron
2. memory
3. channels
4. browser / MCP / tracing
5. multi-tenant / permissions / encryption

目标：最后再碰这些复杂特性。

## 当前默认教学策略

如果用户没有指定要先抄哪个模块，默认从“阶段 1，第 1 步”开始：

- 先写 `go.mod`
- 再写 `main.go`
- 再写 `cmd/root.go`
- 然后补 `cmd/gateway.go`
- 最后补 `pkg/protocol/version.go`

这样做的理由是：

- 它直接对应 `goclaw` 的最外层入口
- 文件少、依赖浅、结构感强
- 用户会先明白“程序从哪进、命令怎么挂、网关怎么起”

## 教学时必须避免的事

- 不要一开始就让用户抄 `internal/agent/` 全量逻辑
- 不要在没有最小占位实现的情况下引用一堆还不存在的包
- 不要把“看起来像最终版”的复杂代码一次性倒给用户
- 不要为了贴近原项目而忽略学习顺序
- 不要默认用户已经理解 Cobra、store、gateway composition root 这些概念

## 每一步要回答的四个问题

1. 这一步在整个系统里的位置是什么
2. 它在 `goclaw` 对应哪个文件或哪一层
3. 用户现在具体要新建哪些文件
4. 写完以后下一步会接到哪里

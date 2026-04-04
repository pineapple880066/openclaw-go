# OpenClaw Go 手写进度

## 基本信息

- 参考项目：`/Users/pineapple/Desktop/OpenClaw_go/goclaw`
- 重写项目：`/Users/pineapple/Desktop/OpenClaw_go/openlaw-go`
- 记录时间：2026-04-02
- 当前状态：已完成 store/sqlite 最小骨架，当前暂停重写，转去验证参考项目 goclaw

## 当前判断

`goclaw` 是一个“大量内部模块 + 极薄入口”的 Go 项目。

手写顺序不能按照源码目录机械地一个文件接一个文件抄，否则会很快陷入：

- 依赖链过深
- 新概念过多
- 文件能抄出来但整体不知道为什么这样接

因此当前采用“从最小主链往里扩”的方式推进。

## 分阶段计划

- [x] 阶段 1：可启动骨架
- [ ] 阶段 2：配置层
- [ ] 阶段 3：基础运行时
- [ ] 阶段 4：核心注册表占位
- [ ] 阶段 5：传输入口
- [ ] 阶段 6：agent loop 最小版
- [ ] 阶段 7：工具系统
- [ ] 阶段 8：存储系统
- [ ] 阶段 9：skills 与 bootstrap
- [ ] 阶段 10：高级能力

## 阶段 1 细分

- [x] 1.1 新建 `go.mod`
- [x] 1.2 新建 `main.go`
- [x] 1.3 新建 `cmd/root.go`
- [x] 1.4 新建 `cmd/gateway.go`
- [x] 1.5 新建 `pkg/protocol/version.go`

## 当前步

- 当前阶段：阶段 2
- 当前步骤：阶段 8 起步，完成 `internal/store/sqlite.go` 最小骨架
- 当前目标：为后续 agent / sessions / telegram 提供持久化地基

## 本步对应 goclaw 源码

- `goclaw/main.go`
- `goclaw/cmd/root.go`
- `goclaw/cmd/gateway.go`
- `goclaw/pkg/protocol/*`

## 为什么先从这里开始

因为这是整个项目最外层、最稳定、最适合建立结构感的一圈：

1. 用户先理解程序入口
2. 用户再理解 Cobra 根命令怎么装
3. 用户再理解真正业务启动点 `runGateway()`
4. 后面再逐步把配置、provider、tool、store 往这个骨架里填

## 下一次教学默认输出

默认只讲“阶段 2”的第一轮代码，按以下顺序给：

1. `internal/config/config.go`
2. `internal/config/load.go`
3. 回到 `cmd/gateway.go` 接入 `config.Load()`
4. `internal/config` 负责路径标准化

## 进度更新规则

每当用户手写完成一个步骤，就把对应项从 `[ ]` 改成 `[x]`，并补充：

- 用户已经写完的文件
- 是否已能 `go run .`
- 当前遇到的编译错误或理解障碍
- 下一步准备进入的文件

## 当前备注

- 已完成最小 CLI 骨架
- 已完成 `config` 模块最小闭环
- 已完成 `internal/store/models.go`、`interfaces.go`、`sqlite.go` 最小骨架
- 2026-04-02 这一天主要完成了 `sqlite.go`
- 当前暂停继续手写，先验证参考项目 `/Users/pineapple/Desktop/OpenClaw_go/goclaw` 是否能在本机编译运行

# 3.2 对应代码说明

这个目录存放 `3.2 把它变成真正的 Chatbot` 的示例代码。

## 文件说明

- `main.go`：程序入口、配置读取、消息历史维护和命令行循环
- `go.mod`：Go 模块定义
- `.env.example`：示例环境变量

## 运行方式

1. 复制配置文件

```bash
cp .env.example .env
```

2. 修改 `.env` 中的模型服务配置

3. 运行程序

```bash
go run .
```

如果你想显式选择响应模式，也可以这样运行：

```bash
go run . -mode sync
go run . -mode stream
```

如果没有传 `-mode`，程序会优先读取 `OPENAI_RESPONSE_MODE`；如果环境变量也没有设置，则默认使用 `stream`。

## 本阶段新增能力

相比 `3.1`，这一版代码新增：

- `system/user/assistant` 三类消息组织
- `messages` 历史列表维护
- 连续输入与退出控制
- 多轮对话上下文承接

## 本阶段边界

这一版仍然不包含：

- 工具调用
- 函数参数 schema
- Agent 调度循环
- Web UI

本阶段的目标只有一个：让读者真正理解“多轮对话是怎么工作的”。

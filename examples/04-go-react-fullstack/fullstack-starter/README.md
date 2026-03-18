# 4.1 全栈骨架示例

这个目录是第 4 章 `AI 全栈技术栈上手（Go + React）` 的对应代码，基于
`go-react-openspec-starter` 模板裁剪而来。

这一版的目标不是马上做 AI 功能，而是先把后面会一直复用的工程骨架跑通：

- 后端怎么分层
- 前端怎么组织请求和类型
- 接口返回格式怎么统一
- 配置应该放在哪里、怎么切环境

如果你能把这一版跑起来，后面继续接模型调用、对话接口、用户系统和业务功能时，
就不会总在“项目到底该怎么搭”这一步反复打转。

## 这一版用了什么技术栈

后端：

- `Go 1.25`
- `Cobra`：命令行入口，统一管理 `server`、`version` 等子命令
- `Gin`：HTTP 路由和交付层
- `Viper`：配置加载，支持 YAML + 环境变量覆盖
- `log/slog`：统一日志输出

前端：

- `React 19`
- `TypeScript`
- `Vite`
- `Axios`：接口请求封装
- `React Router`：页面路由入口

工程约定：

- `OpenSpec`：沉淀后端风格、前端风格、接口规范和测试规范
- `docs/`：把最值得长期复用的工程约定单独放出来，避免只埋在代码里

## 目录结构说明

```text
examples/04-go-react-fullstack/fullstack-starter/
├── backend/
│   ├── cmd/                         # 命令入口，启动服务和查看版本
│   ├── conf/                        # 配置加载与默认值
│   ├── etc/                         # dev / prod 配置文件
│   ├── internal/
│   │   ├── application/sample/      # 应用服务，负责用例编排和 DTO 转换
│   │   ├── domain/sample/           # 领域模型与仓储接口
│   │   ├── infrastructure/storage/  # 基础设施实现，这里先用内存仓储
│   │   ├── interfaces/http/         # HTTP Handler、路由、响应封装
│   │   ├── logging/                 # 日志初始化
│   │   └── wire/                    # 依赖组装
│   ├── Makefile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── components/              # 页面组件和通用组件
│   │   ├── hooks/                   # 通用请求状态 Hook
│   │   ├── services/                # API 请求层
│   │   ├── styles/                  # 页面样式
│   │   └── types/                   # 前端共享类型定义
│   ├── package.json
│   └── vite.config.ts
├── docs/                            # 接口约定、风格约定、测试约定
├── openspec/                        # 规范配置与可复用 spec
└── README.md
```

你可以先把这份结构粗暴记成一句话：

`backend` 负责能力，`frontend` 负责展示，`docs` 和 `openspec` 负责把规则固定下来。

## 后端分层怎么理解

这一版后端不是“按框架文件类型平铺”，而是按职责拆开：

- `domain/`：最核心的业务概念和仓储接口，不关心 HTTP、数据库、前端
- `application/`：把领域对象组织成一个具体用例，对外返回更适合接口层使用的 DTO
- `infrastructure/`：把仓储接口真正落地，这里先放内存实现，后面可以替换成数据库
- `interfaces/http/`：接 HTTP 请求、调应用服务、返回统一 JSON 结构
- `wire/`：把仓储、服务、Handler、Server 串起来，避免所有依赖都散在 `main.go`

这也是后面继续做 AI 功能时最有价值的地方：

- 模型调用客户端可以放进 `infrastructure/`
- 对话用例可以放进 `application/`
- 会话、消息、任务这些核心概念可以沉淀到 `domain/`
- HTTP、WebSocket、SSE 等交付方式仍然待在 `interfaces/`

## 前端分层怎么理解

前端这版也故意不复杂，但有几个很关键的边界：

- `services/api.ts`：统一管理接口调用，不把 `axios.get(...)` 到处散落在页面里
- `types/`：把接口返回结构和页面展示结构显式写出来，减少“字段写错还不自知”
- `hooks/useApi.ts`：统一处理 `loading / error / data` 三态
- `components/`：页面组件只关心展示和交互，不直接拼请求细节

对第 4 章来说，最重要的不是学多少 React 语法，而是先建立一个稳定习惯：
页面组件消费“已经封装好的数据请求”，而不是自己直接去管所有网络细节。

## 接口约定

当前示例先保留两个最小接口：

- `GET /healthz`
- `GET /api/v1/samples`

其中 `/api/v1/samples` 使用统一返回结构：

```json
{
  "code": 0,
  "message": "success",
  "data": []
}
```

这一层约定有三个目的：

- 前端可以稳定判断请求是否成功，而不是每个接口都猜一套格式
- 后端后面加分页、业务错误码、鉴权错误时，不需要推翻整体协议
- 第 5 章往后接测试和规范时，有明确的断言目标

前端默认通过 `frontend/src/services/api.ts` 访问接口：

- `baseURL` 默认取 `VITE_API_BASE_URL`
- 如果没有显式配置，就回退到 `/api/v1`
- Vite 开发环境通过 `vite.config.ts` 里的代理把 `/api` 转发到 `http://localhost:8080`

## 配置方式

后端配置放在：

- `backend/etc/config.dev.yaml`
- `backend/etc/config.prod.yaml`

程序启动时会这样加载：

1. 先根据 `--run-mode` 选择默认配置文件
2. 如果传了 `--configs`，就优先读取指定文件
3. 再用环境变量覆盖 YAML 中的值

比如：

```bash
cd backend
go run ./cmd server --run-mode dev
```

如果你想用环境变量覆盖端口，可以用：

```bash
FULLSTACK_STARTER_HTTP_PORT=9090 go run ./cmd server --run-mode dev
```

这里的规则来自 `Viper`：

- 配置键 `http.port`
- 环境变量前缀 `FULLSTACK_STARTER`
- 点号会自动转成下划线

所以最终就是 `FULLSTACK_STARTER_HTTP_PORT`。

## 如何运行

先启动后端：

```bash
cd backend
go mod tidy
make run
```

后端默认监听 `http://localhost:8080`。

再启动前端：

```bash
cd frontend
npm install
npm run dev
```

前端默认运行在 `http://localhost:3000`。

## 这一版故意不做什么

这一版先不急着塞进下面这些内容：

- 模型调用
- 用户登录
- 数据库持久化
- 多模块业务拆分
- 完整的 AI Agent 编排

因为第 4 章的任务不是“做出一个完整产品”，而是先让读者拿到一个稳定、可扩展、
前后端边界清楚的骨架项目。后面的章节，会在这个骨架上继续往里填真正的 AI 能力。

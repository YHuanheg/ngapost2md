# ngapost2md 项目说明

## 项目概述

ngapost2md 是一个用 Go 语言编写的工具，用于将 NGA 论坛的帖子转换为 Markdown 格式。它能够爬取帖子内容，包括回复人、时间、内容和图片，并将其保存为本地 Markdown 文件。该项目是 2023 年由 Go 语言重写的版本，旧版 Python 版本已不再维护。

主要功能包括：
- 爬取 NGA 论坛帖子内容
- 将内容转换为 Markdown 格式
- 保存帖子中的图片到本地
- 支持只下载特定用户 ID 的发言层
- 支持获取用户 IP 位置
- 支持本地表情图片资源
- 支持文件夹和文件名使用帖子标题
- 支持切分生成的 Markdown 文件
- **Server 模式**：HTTP Server 运行模式，提供 Web 前端界面和 REST API，支持任务队列、定时任务和远程管理

## 项目结构

```
ngapost2md/
├── go.mod          # Go 模块定义文件
├── go.sum          # Go 模块校验和文件
├── main.go         # 程序入口文件
├── config/         # 配置相关代码
│   └── config.go
├── nga/            # 核心功能代码
│   ├── nga.go      # 主要的帖子处理逻辑
│   ├── client.go   # NGA HTTP 客户端（req/v3）
│   └── utils.go    # 工具函数
├── server/         # Server 模式相关代码
│   ├── server.go   # HTTP 服务器、路由、认证中间件
│   ├── session.go  # Session 管理（创建、验证、过期清理，72h TTL）
│   ├── api.go      # API handler 实现（含 login/logout）
│   ├── ws.go       # WebSocket handler 及进度推送
│   ├── task.go     # 任务队列管理（FIFO，单任务执行）
│   ├── schedule.go # 定时任务管理（CRUD + goroutine cron 定时器）
│   ├── frontend.go # 静态文件 serve（go:embed）
│   └── frontend/   # 前端 HTML/CSS/JS 文件（含 login.html）
├── assets/         # 资源文件
│   └── config.ini  # 默认配置文件模板
├── spec/           # 历史设计规格（实现前写就，非现状依据）
│   └── server-mode-spec.md  # Server 模式设计规格
├── docs/           # 现状文档（程序行为以此为准）
│   ├── index.md            # 文档索引与维护约定
│   ├── getting-started.md  # 构建、运行、常见启动问题
│   ├── architecture.md     # 代码结构、数据流、关键机制
│   ├── configuration.md    # config.ini 全部配置项
│   ├── server-api.md       # 认证、REST API、WebSocket
│   └── release.md          # 版本号协同与发布流程
├── README.md       # 项目说明文档
├── AGENTS.md       # 本文件，AI 协作约定
├── README.assets/  # README 引用的截图
└── LICENSE         # 许可证文件
```

## 构建和运行

### 构建环境

Go 1.25.5 安装在本机 `D:\Go`（不在 C 盘）：

- `GOROOT=D:\Go`、`GOPATH=D:\Go\workspace`、`GOENV=D:\Go\goenv\env`，`D:\Go\bin` 已加入用户 PATH
- `GOPROXY=https://goproxy.cn,direct`、`GOSUMDB=sum.golang.google.cn`
- 本机 `hosts` 把 `github.com`、`api.github.com`、`raw.githubusercontent.com` 等域名全部指向 `127.0.0.1`，由本地代理转发；GitHub 可直连，但依赖系统 DNS 的抓取类工具可能因解析到内网地址而被拒
- 若当前 shell 找不到 `go`：`$env:PATH = 'D:\Go\bin;' + $env:PATH`

```bash
go mod download
go build -o ngapost2md main.go
```

### 版本与发布

- 版本号在 `nga/nga.go` 的 `VERSION`，发布时必须打**同名** git tag（如 `2.0.0-fix`），否则 `--update` 会自我误报
- 发布产物由**本地交叉编译**生成，5 个平台：`windows/amd64`、`linux/amd64|arm64`、`darwin/amd64|arm64`（`CGO_ENABLED=0`）
- 包内平铺：可执行文件 + `LICENSE` + `README.md` + `assets/config.ini`，命名沿用 `ngapost2md-NEO_<tag>-<os>-<arch>`
- 上传：`gh release create <tag> --title "ngapost2md ver.[NEO_<tag>]" --notes-file <文件> <产物...>`（本机 `gh` 已登录账号 YHuanheg）
- 本仓库的 GitHub Actions 在 fork 中默认禁用，需在网页端手动启用后才会响应 tag 推送，因此发布不依赖 Actions

### 远端

- `origin` = `https://github.com/YHuanheg/ngapost2md.git`（fork，日常推送目标）
- `upstream` = `https://github.com/ludoux/ngapost2md.git`（上游原仓库）
- 默认分支为 `neo`

### 运行

在运行程序之前，需要先配置 `config.ini` 文件，确保其中的 `ngaPassportUid`、`ngaPassportCid` 和 `ua` 配置项已正确设置。

```bash
# Linux
./ngapost2md <tid>

# Windows
.\ngapost2md.exe <tid>
```

其中 `<tid>` 是 NGA 帖子的 ID。

### 参数说明

- `tid`: 待下载的帖子 ID
- `--authorid <aid>`: 只下载指定用户 ID 的发言层
- `-v, --version`: 显示版本信息并退出
- `-h, --help`: 显示帮助信息并退出
- `-u, --update`: 检查最新版本
- `--gen-config-file`: 生成默认配置文件 `config.ini` 并退出
- `serve [--host ip] [--port port] [--password pwd] [--no-ui]`: 以 Server 模式运行，提供 Web 前端和 HTTP API

### Server 模式

Server 模式以 HTTP Server 运行，提供 Web 前端界面和 REST API：

- **认证**：支持 Session Cookie（Web 前端登录）和 Basic Auth（API 客户端），WebSocket 支持 Session Cookie 或 URL token
- **任务队列**：下载和更新任务通过 FIFO 队列管理，单任务串行执行
- **定时任务**：基于 cron 表达式的定时更新，存储在 `schedules.json`
- **WebSocket 推送**：实时推送下载/更新进度
- **前端嵌入**：纯 HTML/CSS/JS 前端，通过 `go:embed` 嵌入二进制

完整的 REST API 端点表、认证细节与前端页面说明见 [docs/server-api.md](docs/server-api.md) 与 `README.md` 的「Server 模式」一节。`spec/server-mode-spec.md` 是实现前的设计稿（落地后已按最终行为回填），追溯设计意图时参考，**查证行为以 `docs/` 为准**。

Server 模式与 CLI 模式共享 config.ini 和工作目录，互不影响。

### 配置文件

程序需要一个 `config.ini` 文件与可执行文件在同一目录下。配置文件包含网络设置、帖子处理选项等。首次运行前需要修改配置文件中的 `ua`、`ngaPassportUid` 和 `ngaPassportCid` 等项。

配置加载时会与 `config/config.go` 里的 `defaultConfig` 比对：配置项缺失或配置版本变化时，以默认配置为模板回写 `config.ini`（用户自行新增的项会保留），文件不存在时自动生成默认配置。**改动 `defaultConfig` 时必须同步 `assets/config.ini`**，两者是分发包与内置默认值的两份副本，历史上曾因漂移导致用户拿不到新增配置项。

## 开发约定

- 代码使用 Go 语言编写，遵循 Go 语言的编码规范
- 配置文件使用 INI 格式
- 项目使用 Go Modules 进行依赖管理
- 代码结构清晰，功能模块分离（`config`、`nga` 等包）
- 使用 `go-flags` 库处理命令行参数
- 使用 `req/v3` 库进行 HTTP 请求
- 使用 `jsonparser` 库解析 JSON 数据
- 使用 `ants/v2` 库进行并发控制
- 使用 `ini.v1` 库处理 INI 配置文件
- 使用 `cast` 库进行类型转换
- 使用 `gorilla/websocket` 库实现 WebSocket 实时推送（Server 模式）
- 使用 `robfig/cron/v3` 库解析 cron 表达式并执行定时任务（Server 模式）
- 代码中包含详细的注释说明
- 遵循 Go 语言的错误处理模式
- 修改完后只需要测试编译即可，不需要运行测试
- 优先只修改 `*.go`；`server/frontend/` 下的 HTML/CSS/JS 是前端源码（经 `go:embed` 打进二进制），涉及界面缺陷时必须一并修改，不要为了"只动 .go"而绕开
- 不要修改或删除其他非源码文件（`LICENSE`、`README.assets/` 等）
- 有必要时，你需要使用 git 工具来查看前后版本的对比，如使用 git diff 指令
- 假如你需要生成默认配置文件，请先检查当前文件夹下是否存在config.ini，若有请先将它备份，并在生成完毕后再复原

## 深入文档

| 想了解 | 看 |
|---|---|
| 构建、运行与常见启动问题 | [docs/getting-started.md](docs/getting-started.md) |
| 代码结构、数据流、关键机制 | [docs/architecture.md](docs/architecture.md) |
| config.ini 全部配置项 | [docs/configuration.md](docs/configuration.md) |
| REST API 与 WebSocket 字段 | [docs/server-api.md](docs/server-api.md) |
| 版本号协同与发布流程 | [docs/release.md](docs/release.md) |

<!-- neat-freak: initialized at 2026-10-05 -->
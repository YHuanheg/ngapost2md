# 架构

## 两种运行模式

| 模式 | 入口 | 特点 |
|---|---|---|
| **CLI** | `ngapost2md <tid>` | 单帖下载 / 增量更新，前台阻塞执行，进度输出到终端日志 |
| **Server** | `ngapost2md serve` | HTTP 服务，FIFO 任务队列串行执行，WebSocket 推送进度，cron 定时任务 |

两者共享同一份 `config.ini`、同一个工作目录，以及 `nga` 包里的抓取核心。差别只在：任务**怎么被触发**（命令行 vs HTTP / cron）和进度**怎么暴露**（日志 vs WebSocket）。

## 代码结构

| 路径 | 职责 |
|---|---|
| `main.go` | 入口：`serve` 子命令分支、命令行参数解析、启动配置装载、CLI 主流程 |
| `config/config.go` | 默认配置定义、`config.ini` 读写与自动对齐（补缺项 / 版本迁移 / 生成密码） |
| `nga/nga.go` | 抓取与导出核心：分页拉取、楼层解析、内容净化、媒体下载、Markdown 生成、进度回调 |
| `nga/client.go` | NGA HTTP 客户端（`req/v3`，统一 Cookie / UA / 错误处理），包级 `Client`、`BASE_URL`、`UA`、`COOKIE` |
| `nga/utils.go` | 工具：表情映射表、匿名 ID 还原、文件名净化、tid → 文件夹定位、资源下载 |
| `server/server.go` | HTTP 服务器、路由注册、认证中间件、`ScanAllPosts`（扫描本地帖子）、`ReloadConfig` |
| `server/api.go` | REST handler 实现（下载 / 更新 / 任务 / 帖子 / 定时任务 / 配置 / zip 下载） |
| `server/task.go` | 任务队列（FIFO，单任务串行）、任务状态模型、`PostInfo` |
| `server/schedule.go` | 定时任务 CRUD + cron 调度器，持久化到 `schedules.json` |
| `server/session.go` | 登录 Session（72h TTL，Cookie 名 `ngapost2md_session`） |
| `server/ws.go` | WebSocket Hub，广播进度 / 完成 / 失败消息 |
| `server/frontend.go` | `go:embed frontend/*` 静态资源服务（含 SPA 式 fallback 到 `index.html`） |
| `server/frontend/*.html` | 前端页面：`login` / `index`（下载）/ `posts`（帖子列表）/ `schedules` / `config` |
| `assets/config.ini` | 随发布包分发的默认配置模板 |
| `spec/` | 历史设计规格（非现状依据） |
| `docs/` | 现状文档（本目录） |
| `.github/workflows/build.yaml` | tag 触发的 GitHub Actions 发布流程 |

## 启动流程

### CLI（`main.go`）

1. 若 `os.Args[1] == "serve"` → 走 `parseServeArgs` + `server.Start()`，**不经过** go-flags
2. 否则由 go-flags 解析参数；依次处理 `-v` / `-h` / `-u` / `--gen-config-file`（均直接 `os.Exit`）
3. `config.GetConfigAutoUpdate()`：加载 `config.ini`，补缺项、做版本迁移、必要时回写
4. 校验 `ua` / `ngaPassportUid` / `ngaPassportCid` 非空且不含 `MODIFY_ME`
5. `nga.ApplyConfig()` 把配置写入 `nga` 包级变量；校验互斥项（`enhance_ori_reply` 与 `thread`）
6. `nga.NewNgaClient()`
7. `FindFolderNameByTid`：本地已有该帖 → `InitFromLocal`（增量）；否则 `InitFromWeb`（新建）
8. `tie.Download()`

### Server（`server.Start`）

1. `GetConfigAutoUpdate()`；确定密码：`--password` > `[server].password`，都为空则**生成随机密码、写入 config.ini、打印到终端**
2. 确定 `host` / `port`：命令行 > 配置 > 默认（`0.0.0.0` / `8080`）
3. 装载 Cookie / UA / base_url 到 `nga` 包，`ApplyConfig`，互斥校验，`NewNgaClient()`
4. 构造 `SessionManager` → `WebSocketHub` → `TaskManager` → `ScheduleManager`（加载并启动 cron）
5. 注册路由 + 认证中间件 → `http.ListenAndServe`

## 数据流

### 抓取与导出（CLI 与 Server 共用）

```
InitFromWeb / InitFromLocal
   └─ page(page)            拉取单页 → 写入 Tiezi 元信息（标题/作者/总页数/楼层数）
                             ↓ 单次下载上限在此裁剪 WebMaxPage
Floors.analyze               解析 JSON 楼层 → 填充 Floors
fixFloorContent              内容净化：引用、链接、表情、匿名、骰子、图片/音视频、IP 归属地
processMedia                 按配置下载媒体到本地或保留在线链接
genMarkdown                  post.md / {标题}.md（或切分 post-001.md…）
SaveProcessInfo / SaveAssetsMap   写回 process.ini / assets.json
```

### Server 任务链路

```
HTTP POST /api/download | /api/update | cron 触发
   └─ TaskManager.Enqueue*        入队（同一 tid 已在队列 / 执行中 → 409）
        └─ processQueue            FIFO，取队首，标记 downloading
             └─ executeTask       构造 nga.Tiezi，绑定 ProgressCallback
                  └─ tie.Download()
                       └─ ProgressCallback(stage, page, floor…)
                            └─ TaskManager 更新 TaskStatus → WebSocketHub.Broadcast
                                                                  └─ 前端刷新进度
```

**并发约束**：任务由 `TaskManager` 串行执行，任意时刻只有一个 `Tiezi` 在跑。任务**不能取消**（`nga` 层没有取消机制），只能移除队列中尚未开始的任务。

## 核心数据结构

### `nga.Tiezi`（一次抓取任务的完整状态）

| 字段 | 含义 |
|---|---|
| `Tid` / `AuthorId` | 帖子 ID / 只看某用户（0 表示全部） |
| `Title` / `TitleFolderSafe` / `Catelogy` | 标题 / 净化后的标题 / 分区 |
| `WebMaxPage` | 本次要抓到的页数（可能已被单次下载上限裁剪） |
| `WebRealMaxPage` | NGA 报告的真实总页数（未裁剪），用于判断是否已完整下载 |
| `LocalMaxPage` / `LocalMaxFloor` | 本地进度（来自 `process.ini`） |
| `FloorCount` / `Floors` / `HotPosts` | 楼层总数（含主楼）/ 楼层数组（下标即楼层号）/ 热门回复 |
| `PageDownloadLimit` / `PageDownloadLimitTriggered` | 本次生效的页数上限 / 是否因上限被截断 |
| `ProgressCallback` | Server 模式注入的进度回调；CLI 模式为 nil |

> `PageDownloadLimit*` 是**每任务实例状态**。历史上它们是包级全局变量，导致 Server 模式下多任务之间互相串味，现已收敛到 `Tiezi`。

### `server.TaskStatus`

`queued` → `downloading` → `processing` → `generating_markdown` → `completed` / `failed`（`cancelled` 仅用于队列中被移除的任务）。

达到单次下载页数上限时任务**仍然是 `completed`**，但会带上 `limited=true`、`webTotalPage`、`pageDownloadLimit`，前端据此提示"未下载完整"。

### `server.PostInfo`（`/api/posts` 的返回项）

`tid`、`authorId`、`title`、`folderName`、`maxPage`、`maxFloor`、`webMaxPage`、`hasMarkdown`、`createdTime`、`updatedTime`。

`webMaxPage > maxPage` 表示该帖尚未下载完整。

## 磁盘产物

每个帖子一个文件夹（`[post].output_path/<folderName>/`）：

| 文件 | 内容 |
|---|---|
| `post.md` 或 `{标题}.md` | 导出的 Markdown；开启切分时为 `post-001.md`、`post-002.md`… |
| `process.ini` | `[local] max_floor` / `max_page` / `web_max_page`，`[info] created_time` / `updated_time` |
| `assets.json` | 媒体在线 URL → 本地文件名映射 |
| `assets/` | 下载到本地的图片 / 音频 / 视频（受 `assets_path`、`use_network_media_url` 影响） |
| `splitinfo.ini` | 切分模式的状态（当前分片序号、剩余楼层数） |

程序级文件：`config.ini`（配置）、`schedules.json`（定时任务，Server 模式生成）。

## 关键机制

**增量更新**
本地已存在该帖文件夹时走 `InitFromLocal`：从 `process.ini` 读回 `max_page` / `max_floor`，只抓取之后的页面与楼层，Markdown 以追加方式写入。

**单次下载页数上限（#56）**
`[network].page_download_limit`（默认 100，-1/0 不限制）。在 `page()` 内把 `WebMaxPage` 裁剪为 `LocalMaxPage + limit`，这样 `Download()` 的分页循环边界自然收敛；同时在 `Tiezi` 上记录是否触发及真实总页数。**这就是"一次跑不完、需要反复运行"的机制来源。**

**并发模型**
`ants/v2` 协程池 + `[network].thread`（仅 1/2/3）+ `DELAY_MS=330ms` 的请求间隔。`enhance_ori_reply` 会额外发起大量请求，因此要求 `thread=1`。

**媒体处理**
默认把图片/音视频下载到本地并改写 Markdown 引用（`use_network_media_url=true` 则保留在线链接）；`cookies.json`、`config.ini` 被 `.gitignore` 忽略。

**表情**
`[s:ac:xx]` 短代码经 `nga/utils.go` 的内置映射表转成文件名；`use_local_smile_pic=true` 时引用 `local_smile_pic_path` 下的本地资源，否则引用在线资源。

**配置自愈**
`config.GetConfigAutoUpdate()` 会把 `config.ini` 与内置默认配置对齐：缺项补全并回写、版本变化时迁移并回写、用户已有值与自定义项保留。`config.ini` 整个文件不存在时自动生成默认配置。详见 [configuration.md](configuration.md)。

**前端嵌入**
前端 HTML 通过 `go:embed` 编进二进制。**改完 `server/frontend/*.html` 必须重新 `go build` 才会生效**，直接替换磁盘上的 HTML 文件不影响已编译的程序。

## 已知实现缺陷（待修）

以下现象与设计意图不一致。**修代码，不要按现状改文档**——这里的描述保留的是"应该怎样"。

| 位置 | 现象 |
|---|---|
| `server.ScanAllPosts` | `PostInfo.FloorCount` 字段已定义但从未赋值，导致 `/api/posts` 永远不会返回 `floorCount`（被 `omitempty` 省略） |
| `nga` 包配置 | `CFGFILE_*` 是进程级全局变量，`PUT /api/config` 会**立即影响正在执行的任务**；`spec/server-mode-spec.md` 中"正在运行的任务使用任务启动时的配置"的约定目前并不成立 |
| `DELETE /api/tasks/{tid}` | 取消"正在执行中"的任务返回 404，与"任务不存在"同码，语义上更接近 409 |


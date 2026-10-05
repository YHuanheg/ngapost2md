# Server 模式与 HTTP API

以 `ngapost2md serve` 启动，提供 Web 前端与 REST API。

```bash
./ngapost2md serve [--host 0.0.0.0] [--port 8080] [--password <密码>] [--no-ui]
```

启动后访问 `http://<host>:<port>/`，用户名固定 `admin`，密码取 `--password` 或 `[server].password`（都为空时会自动生成并写入 config.ini，同时打印在终端）。`--no-ui` 时不服务前端页面，只留 API。

## 认证

认证由 `authMiddleware` 统一处理，优先级为 **Session Cookie > Basic Auth**。

| 方式 | 适用 | 说明 |
|---|---|---|
| Session Cookie | Web 前端 | `POST /api/login` 成功后下发 `ngapost2md_session`（HttpOnly、SameSite=Lax、Path=/），有效期 72 小时，服务端每小时清理过期会话 |
| Basic Auth | API 客户端 | 用户名 `admin`，密码同上 |
| WebSocket | 浏览器 / 客户端 | 优先 Session Cookie；也支持 `GET /ws?token=base64(admin:<password>)` |

**公开路由**（无需认证）：`POST /api/login`、`POST /api/logout`、`GET /api/version`、`GET /login.html`、`GET /ws`（`/ws` 在 handler 内部自行校验）。

其余请求：API 返回 401，页面请求（GET 且非 `/api/`）302 重定向到 `/login.html`。

## REST API

| Method | Path | 请求体 / 参数 | 说明 |
|---|---|---|---|
| `POST` | `/api/login` | `{"username":"admin","password":"..."}` | 登录，成功下发 session cookie |
| `POST` | `/api/logout` | — | 登出，删除 session 并清 cookie |
| `GET` | `/api/version` | — | `{"version":"2.0.0-fix"}` |
| `POST` | `/api/download` | `{"tid":123456,"authorId":0}` | 新帖下载入队，返回 202 + 任务对象 |
| `POST` | `/api/update` | `{"tid":123456}` | 增量更新入队，返回 202 + 任务对象 |
| `GET` | `/api/tasks` | — | 当前运行中的任务 + 队列，空队列返回 `[]` |
| `DELETE` | `/api/tasks/{tid}` | — | 移除**队列中**的任务 |
| `GET` | `/api/posts` | — | 已下载帖子列表 |
| `GET` | `/api/posts/{tid}/download` | — | 把帖子文件夹打包为 zip 流式下载 |
| `DELETE` | `/api/posts/{tid}` | — | 删除帖子文件夹（不可恢复） |
| `GET` | `/api/schedules` | — | 定时任务列表 |
| `POST` | `/api/schedules` | `{"tid":123456,"authorId":0,"cron":"0 9 * * 1-5"}` | 创建，返回 201 |
| `PUT` | `/api/schedules/{id}` | `{"cron":"...","enabled":true}` | 更新 |
| `DELETE` | `/api/schedules/{id}` | — | 删除 |
| `GET` | `/api/config` | — | 全部配置（不含 `[server].password`），附 `_comments` 注释 |
| `PUT` | `/api/config` | `{"post":{"use_title_as_folder_name":"True"}}` | 局部更新并立即生效 |
| `GET` | `/ws` | — | WebSocket 实时进度 |

### 任务对象的字段

```json
{
  "tid": 234567,
  "authorId": 0,
  "type": "update",
  "status": "completed",
  "currentPage": 3, "totalPage": 3,
  "currentFloor": 178, "totalFloor": 178,
  "stage": "completed",
  "queueTime": "2026-10-05T21:20:00+08:00",
  "startTime": "2026-10-05T21:20:05+08:00",
  "endTime": "2026-10-05T21:21:30+08:00",
  "limited": true,
  "webTotalPage": 9,
  "pageDownloadLimit": 100
}
```

- `status`：`queued` → `downloading` → `processing` → `generating_markdown` → `completed` / `failed`；`cancelled` 仅用于被移除的排队任务
- `stage`：`downloading` / `processing_content` / `generating_markdown` / `completed`
- `limited=true` 表示**达到了单次下载页数上限**：任务本身成功，但帖子并未下载完整，`webTotalPage` 是 NGA 报告的真实总页数。再次调用 `/api/update` 可继续下载后续页面
- 任务完成后即从队列中移除，`GET /api/tasks` 不再返回它

### 帖子列表对象的字段

```json
{
  "tid": 234567, "authorId": 0,
  "title": "帖子标题", "folderName": "234567-帖子标题",
  "maxPage": 3, "maxFloor": 178, "webMaxPage": 9,
  "hasMarkdown": true,
  "createdTime": "2026-05-01T10:00:00+08:00",
  "updatedTime": "2026-05-02T11:30:00+08:00"
}
```

`webMaxPage > maxPage` 表示该帖尚未下载完整。`hasMarkdown` 的判定：存在 `post.md` / `{文件夹名}.md` / `splitinfo.ini` 之一。

### 错误语义

| 状态码 | 场景 |
|---|---|
| 400 | `tid` 为 0 或缺失、请求体非法、cron 表达式为空、试图修改 `[server].password` |
| 401 | 未认证（API）；页面请求则 302 到登录页 |
| 404 | 取消不存在或**正在执行中**的任务、删除不存在的帖子文件夹、更新/删除不存在的定时任务 |
| 409 | 同一 `tid` 已在队列或正在执行 |
| 500 | 配置加载 / 保存失败等内部错误 |

## WebSocket

连接 `ws://<host>:<port>/ws`。消息为 JSON，三种 `type`：

```json
{"type":"progress","tid":234567,"taskType":"update","status":"downloading",
 "currentPage":2,"totalPage":3,"currentFloor":40,"totalFloor":178,"stage":"downloading"}
```

```json
{"type":"task_complete","tid":234567,"taskType":"update","status":"completed",
 "limited":true,"webTotalPage":9,"pageDownloadLimit":100,
 "message":"已达单次下载页数上限（100 页）：本次已下载至第 3 页，全帖共 9 页，尚未下载完整。请再次执行增量更新以继续下载后续页面。"}
```

```json
{"type":"task_failed","tid":234567,"taskType":"download","status":"failed","error":"下载页面失败: ..."}
```

`limited` 为 `true` 时，前端会显示红色告警块并提供「继续增量更新此帖」按钮。服务端每 30 秒发送一次 ping 保活。

## 前端页面

| 页面 | 路径 | 功能 |
|---|---|---|
| 登录 | `/login.html` | 用户名（默认 admin）+ 密码；成功后跳转 `/` |
| 下载帖子 | `/index.html`（`/` 亦指向此页） | 输入 tid 或 NGA URL 入队；队列表格；阶段/页面/楼层进度条；达到单次上限时提示并可一键继续更新 |
| 帖子列表 | `/posts.html` | 已下载帖子表格；未下载完整的帖子会标注「全帖 N 页，未下载完整」；支持增量更新、打包下载、删除 |
| 定时任务 | `/schedules.html` | 定时任务的增删改启停，附常用 cron 模板按钮 |
| 配置 | `/config.html` | 分组展示 `config.ini`，悬停显示注释，保存后即时生效（`[server].password` 不显示） |

前端是**薄客户端**：队列状态一律以后端 `GET /api/tasks` 为准，页面只负责渲染与转发；实时进度依赖 WebSocket，收到消息后刷新界面。

> 前端 HTML 通过 `go:embed` 编入二进制：**改动后必须重新 `go build`**，直接替换磁盘上的 HTML 不会生效。

## 安全提示

- Server 模式**没有 HTTPS**，`password` 与 Cookie 以明文传输，请勿直接暴露在公网；需要外网访问时请置于反向代理（并配置 TLS）之后
- 登录接口无频率限制，弱密码有被暴力破解的风险，建议使用自动生成的长随机密码
- `DELETE /api/posts/{tid}` 会直接删除磁盘上的帖子文件夹，不可恢复

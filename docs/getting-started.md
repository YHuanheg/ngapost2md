# 快速开始

## 环境要求

| 项目 | 要求 |
|---|---|
| Go | 1.25.0 及以上（`go.mod` 声明 `go 1.25.0`，`toolchain go1.25.5`） |
| 网络 | 能访问 NGA 论坛，默认 `https://bbs.nga.cn`（可在配置中改） |
| NGA 凭据 | 浏览器 Cookie 中的 `ngaPassportUid` 与 `ngaPassportCid` |

项目**不使用任何环境变量**，全部配置通过 `config.ini` 完成。

## 获取代码与依赖

```bash
git clone https://github.com/YHuanheg/ngapost2md.git
cd ngapost2md
go mod download
```

## 构建

```bash
go build -o ngapost2md main.go
```

发布包会通过 ldflags 注入构建信息（本地调试可省略，省略时 `-v` 显示的是源码里的默认值）：

```bash
go build -ldflags "\
  -X github.com/ludoux/ngapost2md/nga.DEBUG_MODE=0 \
  -X github.com/ludoux/ngapost2md/nga.BUILD_TS=$(date +%s) \
  -X github.com/ludoux/ngapost2md/nga.GIT_REF=refs/tags/2.0.0-fix \
  -X github.com/ludoux/ngapost2md/nga.GIT_HASH=$(git rev-parse HEAD)" \
  -o ngapost2md main.go
```

> `DEBUG_MODE=1`（源码默认值）时 `main.go` 会额外打印参数解析结果，不影响功能。

更完整的交叉编译与发布流程见 [release.md](release.md)。

## 准备 config.ini

程序从**工作目录**读取 `config.ini`（需与可执行文件在同一目录）。三种获取途径：

1. 从 Release 压缩包解压（包内已附带）
2. 执行 `./ngapost2md --gen-config-file` 生成默认配置 —— 会**覆盖**已有的 `config.ini`
3. 直接运行程序：若当前目录没有 `config.ini`，程序会**自动生成**一份默认配置并退出，提示你填写后再运行

必须填写的三项（未填写或仍是占位符会直接报错退出）：

```ini
[network]
ua=`<你的浏览器 UA>`
ngaPassportUid=`<你的 uid>`
ngaPassportCid=`<你的 cid>`
```

Cookie 获取方式：浏览器登录 NGA → 开发者工具 → Application/存储 → Cookies → 复制 `ngaPassportUid` 与 `ngaPassportCid` 两项的值。

> 值建议用反引号包裹（例如 ``ua=`Mozilla/5.0 ...` ``）。INI 解析器会处理转义，未被包裹的值中含特殊字符时可能被截断。

程序启动时会自动把 `config.ini` 与内置默认配置对齐：缺失的配置项会被补全并回写文件，已有的值和用户自行新增的配置项都会保留。详见 [configuration.md](configuration.md)。

## CLI 用法

```bash
./ngapost2md <tid>                      # 下载帖子
./ngapost2md <tid> --authorid <aid>     # 只导出某用户的楼层
./ngapost2md "https://nga.178.com/read.php?tid=123&authorid=456"   # 直接贴链接
./ngapost2md -v                         # 显示版本与构建信息
./ngapost2md -h                         # 显示帮助
./ngapost2md -u                         # 检查更新（查询本仓库 YHuanheg/ngapost2md）
./ngapost2md --gen-config-file          # 生成默认 config.ini（覆盖既有文件）
```

对同一个 tid **再次运行即为增量更新**：程序读取本地 `process.ini` 记录的进度，只抓取新页面与新楼层。

> Windows 用户可以省掉敲命令：双击包内的 `win_CLICK_ME_TO_START.bat`，出现菜单后按 `[1]` 输入 tid 下载、按 `[2]` 按用户筛选、按 `[3]` 启动 Server 模式、按 `[4]` 用记事本编辑 `config.ini`。

输出位置由 `[post].output_path` 决定（默认 `./`）；文件夹名由 `[post].use_title_as_folder_name` 决定，关闭时为 `123456`，开启时为 `123456-帖子标题`；只看某用户时为 `123456(789)` / `123456(789)-帖子标题`。

## Server 模式

```bash
./ngapost2md serve [--host 0.0.0.0] [--port 8080] [--password <密码>] [--no-ui]
```

- 未指定 `--password`，且 `config.ini` 的 `[server].password` 为空或仍是占位符时，程序会**自动生成密码**、写入 `config.ini` 并打印在终端
- 浏览器访问 `http://<host>:<port>`，用户名固定为 `admin`
- `--no-ui` 只提供 API，不服务前端页面
- Server 模式与 CLI 模式共享同一个 `config.ini` 和工作目录

页面、认证与 API 详见 [server-api.md](server-api.md)。

## 冒烟验证

```bash
./ngapost2md -v                          # 能打印版本 / 构建时间 / Git 信息
./ngapost2md serve --port 18080          # 能启动，浏览器能打开登录页
curl -u admin:<密码> http://127.0.0.1:18080/api/version   # Basic Auth 可用
```

> 验证 `--gen-config-file` 前请先备份现有 `config.ini`，它会覆盖文件。

## 常见启动失败

| 现象 | 原因 |
|---|---|
| `配置项配置错误: ua=` | `[network].ua` 为空或仍是 `<;MODIFY_ME;>` |
| `配置项配置错误: ngaPassportUid=` / `ngaPassportCid=` | 未填写 Cookie，或值仍含 `MODIFY_ME` |
| `无法加载配置文件` | `config.ini` 存在但无法解析（损坏、编码或权限问题）；文件**不存在**时程序会自动生成默认配置，不会报此错 |
| `配置项互斥检查失败，请检查 enhance_ori_reply thread` | 开启 `enhance_ori_reply` 时必须把 `thread` 设为 1 |
| `配置项互斥检查失败，请检查 enhance_ori_reply_online enhance_ori_reply` | 开启在线增强前必须先开启 `enhance_ori_reply` |
| Server 启动后忘记密码 | 密码写在 `config.ini` 的 `[server].password`，或见启动日志 |

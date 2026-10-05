# 配置说明（config.ini）

程序只从**工作目录**下的 `config.ini` 读取配置，不使用环境变量。`config.ini` 必须与可执行文件同级，否则启动即报 `无法加载配置文件`。

## 自动对齐行为

每次启动（CLI 与 Server 相同）都会用内置默认配置对齐现有文件：

| 情况 | 行为 |
|---|---|
| 配置文件不存在 | 自动生成一份完整默认配置并退出，提示填写 `ua` / `ngaPassportUid` / `ngaPassportCid` |
| 缺少默认配置里的某个键 | 自动补齐该键（保留注释）并回写文件，日志打印补了哪些项 |
| `[config].version` 与程序内置版本不同 | 执行迁移、整体回写并打印版本变更 |
| 已有键 | 保留用户的值 |
| 用户自行新增的 section / key | **保留**（回写时会一并写回） |
| 文件损坏 / 缺少 `[config].version` | 拒绝加载并报错，不会覆盖文件 |

> 因此：把发布包里附带的旧 `config.ini` 直接拿来用是安全的，程序会自己补齐缺项；但**不建议在 `config.ini` 里写自定义注释**，回写时可能丢失。

`--gen-config-file` 会用默认配置**覆盖**当前文件；执行前请自行备份。

## `[config]`

| 键 | 默认值 | 说明 |
|---|---|---|
| `version` | `2.0.0` | 配置文件格式版本，由程序维护，**不要手改**。它跟随的是配置结构，而非程序发布版本号 |

## `[network]`

| 键 | 默认值 | 说明 |
|---|---|---|
| `base_url` | `https://bbs.nga.cn` | 访问的 NGA 域名 |
| `ua` | `<;MODIFY_ME;>` | 浏览器 User-Agent，**必填**，建议用反引号包裹 |
| `ngaPassportUid` | `<;MODIFY_ME;>` | NGA Cookie 项，**必填** |
| `ngaPassportCid` | `<;MODIFY_ME;>` | NGA Cookie 项，**必填** |
| `thread` | `2` | 网络并发数，仅接受 1 / 2 / 3。开启 `enhance_ori_reply` 时必须为 1 |
| `page_download_limit` | `100` | 单次运行最多新下载的页数，取值范围 -1 ~ 100；`0` 或 `-1` 表示不限制。达到上限后需再次运行以继续 |

## `[post]`

| 键 | 默认值 | 说明 |
|---|---|---|
| `get_ip_location` | `False` | 查询用户 IP 归属地。启用后网络请求量最多增加 20 倍 |
| `enhance_ori_reply` | `False` | 补全被引用楼层的原文。要求 `thread=1` 且需全新拉取 |
| `enhance_ori_reply_online` | `False` | 在线增强原始回复，需先开启 `enhance_ori_reply` |
| `use_local_smile_pic` | `False` | 使用本地表情资源替代在线引用 |
| `local_smile_pic_path` | `../smile/` | 本地表情目录，末尾需带 `/` |
| `use_title_as_folder_name` | `False` | 文件夹名使用帖子标题。**仅对全新拉取的 tid 生效** |
| `use_title_as_md_file_name` | `False` | Markdown 文件名使用帖子标题。**仅对全新拉取的 tid 生效** |
| `use_network_media_url` | `False` | 媒体只引用在线链接，不下载到本地 |
| `assets_path` | `./assets/` | 媒体存放路径，相对帖子文件夹。留空或 `.` 表示与输出文件同目录 |
| `split_md_file` | `-1` | 切分 Markdown，值为每个文件约含页数（1 页 ≈ 20 楼），范围 -1 ~ 200；`0` 或 `-1` 不切分 |
| `output_path` | `./` | 帖子输出根目录，支持绝对路径与相对路径 |

## `[server]`

| 键 | 默认值 | 说明 |
|---|---|---|
| `host` | `0.0.0.0` | 绑定 IP，可被 `--host` 覆盖 |
| `password` | `<;MODIFY_ME;>` | 登录 / Basic Auth 密码。为空或仍是占位符时，启动时自动生成并写回本文件 |
| `port` | `8080` | 监听端口，可被 `--port` 覆盖 |

`[server].password` 不允许通过 `PUT /api/config` 修改（安全考虑），只能手改文件或用 `--password`。

## 互斥与校验规则

启动时按顺序校验，任一不满足即退出：

1. `ua`、`ngaPassportUid`、`ngaPassportCid` 非空且不含 `MODIFY_ME`
2. `enhance_ori_reply=true` 时，`thread` 必须为 1
3. `enhance_ori_reply_online=true` 时，`enhance_ori_reply` 必须为 true

数值型配置越界时会被夹到合法区间（例如 `thread=5` → 取允许值，`split_md_file=999` → 200）。

## 典型配置示例

### 保守：只导文本，不下载媒体

```ini
[post]
use_network_media_url=True
get_ip_location=False
```

### 完整存档：下载图片 + 按标题命名 + 切分

```ini
[network]
thread=2
page_download_limit=100

[post]
use_title_as_folder_name=True
use_title_as_md_file_name=True
split_md_file=50
output_path=./archive/
```

### 补全引用原文（慢，请求量大）

```ini
[network]
thread=1

[post]
enhance_ori_reply=True
enhance_ori_reply_online=True
```

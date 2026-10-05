# ngapost2md 文档索引

ngapost2md 把 NGA 论坛帖子导出为 Markdown。它有两种运行方式：**CLI**（单帖下载 / 增量更新）和 **Server 模式**（HTTP API + WebUI + 任务队列 + 定时任务）。

本目录（`docs/`）描述**当前实现**，是查证程序行为时的权威来源。

## 阅读路径

| 我想…… | 看这里 |
|---|---|
| 构建并跑起来 | [getting-started.md](getting-started.md) |
| 搞懂代码怎么组织、数据怎么流动 | [architecture.md](architecture.md) |
| 查 `config.ini` 每一项配置 | [configuration.md](configuration.md) |
| 对接 HTTP API / WebSocket | [server-api.md](server-api.md) |
| 发一个版本 | [release.md](release.md) |
| 看项目介绍与使用说明 | [../README.md](../README.md) |
| 看 AI 协作约定与红线 | [../AGENTS.md](../AGENTS.md) |

## 目录约定

| 位置 | 定位 |
|---|---|
| `docs/` | **现状文档**，随代码更新，以这里为准 |
| `spec/` | 历史设计规格。`spec/server-mode-spec.md` 写于 Server 模式实现**之前**，实现过程中有多处调整，仅供追溯设计意图，**不作为行为依据** |
| `README.md` | 面向使用者的项目介绍与上手说明 |
| `AGENTS.md` | 面向 AI 协作者的项目约定与红线 |

## 维护约定

改了行为就顺手改文档，别让文档落后于代码：

| 改动 | 需要同步 |
|---|---|
| 新增 / 修改配置项 | [configuration.md](configuration.md) + [assets/config.ini](../assets/config.ini) + [config/config.go](../config/config.go) 中的默认值 |
| 新增 / 修改 API 路由 | [server-api.md](server-api.md) + [README.md](../README.md) 的 REST API 表 |
| 包结构变化 | [architecture.md](architecture.md) + [AGENTS.md](../AGENTS.md) 的项目结构 |
| 发布流程变化 | [release.md](release.md) |
| 版本号 bump | [nga/nga.go](../nga/nga.go) 的 `VERSION` + [README.md](../README.md) 标题 + git tag，三者必须一致（见 [release.md](release.md)） |

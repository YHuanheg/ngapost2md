# 发布流程

## 版本号：三处必须一致

| 位置 | 作用 |
|---|---|
| `nga/nga.go` 的 `VERSION` | 程序内版本号，`-v` 显示、`--update` 比对 |
| git tag | 触发 GitHub Actions、决定 Release 名与产物文件名 |
| `README.md` 标题 | 面向读者的版本标识 |

**三者不一致会发生什么**：`--update` 拿 `VERSION` 与 GitHub Releases 的 `tag_name` 比较，不相等就提示"请去下载最新版本" —— 即使你用的就是最新版。

> `config.ini` 里的 `[config].version`（`config/config.go` 中 `defaultConfig`）**不是**程序版本，而是**配置文件格式版本**。配置结构没变时不要跟着 bump；它变化会触发所有用户的配置文件迁移。

### bump 步骤

```bash
# 1. 改版本号
#    nga/nga.go: VERSION = "2.0.1"
#    README.md : # ngapost2md ver.[NEO_2.0.1]
git add nga/nga.go README.md
git commit -m "chore: bump version to 2.0.1"
git push origin neo
```

## 发布前检查

```bash
go build -o ngapost2md main.go     # 必须通过
go vet ./...                       # 必须无输出
./ngapost2md -v                    # 版本号是否为新值
```

> 本仓库的换行符是 CRLF（`core.autocrlf=true`），`gofmt -l .` 会把所有文件都标为"待格式化"——这是环境噪声。要判断格式是否真的有问题，先把文件按 LF 归一化再跑 `gofmt -l`。

平台矩阵与产物命名沿用上游约定：

| OS | Arch | 压缩格式 |
|---|---|---|
| windows | amd64 | `.zip` |
| linux | amd64 / arm64 | `.tar.gz` |
| darwin | amd64 / arm64 | `.tar.gz` |

产物名：`ngapost2md-NEO_<tag>-<os>-<arch>`；包内**平铺**（`assets/` 下的文件不带目录前缀）：可执行文件、`LICENSE`、`README.md`、`config.ini`、`win_CLICK_ME_TO_START.bat` + `win_CLICK_ME_TO_START.ps1`（Windows 双击启动器，菜单式，含 CLI 与 Server 两种模式）、`win_CLICK_ME_TO_CHECK_UPDATE.bat` + `win_updater.ps1`（Windows 更新入口）。按上游约定，这些脚本所有平台包都放。

> ⚠️ 打包时 `assets/*` 要**遍历整个 `assets/` 目录**，不要只复制 `config.ini`；且 `config.ini` 必须与可执行文件**同级**——这是 README 对用户的承诺（"config.ini 文件与主程序平级"）。若把 `config.ini` 塞进 `assets/` 子目录，用户解压后程序在根目录找不到配置，只会重新生成一份默认配置。
>
> ⚠️ `win_CLICK_ME_TO_START.ps1` **必须保存为 UTF-8 with BOM**：Windows PowerShell 5.1 读取无 BOM 的 UTF-8 时会按 ANSI 解码，中文界面会变成乱码。配套的 `.bat` 只做转发，**必须保持纯 ASCII**——cmd.exe 会用当前代码页解码整个 .bat 文件，`chcp 65001` 来不及生效，含中文的 .bat 在中文系统上会被拆成乱码命令。

> ⚠️ 打包时 `assets/*` 要**遍历整个 `assets/` 目录**，不要只复制 `config.ini`；且 `config.ini` 必须与可执行文件**同级**——这是 README 对用户的承诺（"config.ini 文件与主程序平级"）。若把 `config.ini` 塞进 `assets/` 子目录，用户解压后程序在根目录找不到配置，只会重新生成一份默认配置。

## 本地交叉编译（当前采用的方式）

```bash
export CGO_ENABLED=0
TAG=2.0.1
TS=$(date +%s)
HASH=$(git rev-parse "refs/tags/$TAG^{commit}")   # 取 tag 指向的提交，不要用 HEAD
LD="-X github.com/ludoux/ngapost2md/nga.DEBUG_MODE=0 \
    -X github.com/ludoux/ngapost2md/nga.BUILD_TS=$TS \
    -X github.com/ludoux/ngapost2md/nga.GIT_REF=refs/tags/$TAG \
    -X github.com/ludoux/ngapost2md/nga.GIT_HASH=$HASH"

GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$LD" -o ngapost2md.exe main.go
GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "$LD" -o ngapost2md      main.go
GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "$LD" -o ngapost2md      main.go
GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "$LD" -o ngapost2md      main.go
GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "$LD" -o ngapost2md      main.go
```

每个平台把二进制 + `LICENSE` + `README.md` + `assets/config.ini` 放到独立目录后压缩。

**发布前必须做一次冒烟测试**（交叉编译产物无法直接运行非本机平台，至少验证本机平台）：

```bash
./ngapost2md.exe -v
# ngapost2md 2.0.1
# Build_Time: ...  Git_Ref: refs/tags/2.0.1  Git_Hash: <sha>
```

`Git_Ref` 与 `Git_Hash` 正确，说明 ldflags 注入成功、产物不是空的。

## 发布

```bash
git tag -a 2.0.1 -m "ngapost2md ver.[NEO_2.0.1]"
git push origin 2.0.1

gh release create 2.0.1 \
  --repo YHuanheg/ngapost2md \
  --title "ngapost2md ver.[NEO_2.0.1]" \
  --notes-file release_notes.md \
  <产物文件...>
```

## GitHub Actions

`.github/workflows/build.yaml` 在 **tag push** 时触发两个 job：生成 changelog 创建 Release，然后用 `go-release-action` 编译矩阵并上传产物。

**在 fork 中默认不生效**：GitHub 会禁用 fork 的 workflow，需要在网页端 Actions 页面点一次 "I understand my workflows, go ahead and enable them" 之后才会响应推送。此外 workflow 的 token 权限需要是 **Read and write**（Settings → Actions → General → Workflow permissions），否则创建 Release 会 403。

因此当前的发布流程是**本地交叉编译 + `gh release create`**，不依赖 Actions。启用 Actions 后可以改回由 tag 自动发布，注意两者不要同时创建同一个 Release。

## 本 fork 的仓库指向约定

程序内**对外可见**的仓库地址全部指向本 fork（`YHuanheg/ngapost2md`）：

- `--update`：查询 `https://api.github.com/repos/YHuanheg/ngapost2md/releases/latest`；程序内 `VERSION` 与 Release tag 一致时输出"当前已是最新版本"
- 启动 banner、`-h` 帮助首行、生成的 Markdown 页脚、WebUI 导航栏的 GitHub 图标、定时任务页的 Issues 链接
- `assets/win_updater.ps1` 的 API 与 changelog 链接

**刻意不改**的两处：

- `go.mod` 的 module 路径与代码里的 import 仍是 `github.com/ludoux/ngapost2md/...`。它是 Go 包标识，改动需要同步全部 import 与所有 `-ldflags` 的 `-X` 参数，且会让本仓库无法再从上游 merge；它不出现在任何用户可见的信息里
- README 中 `[#109]` / `[#103]` 这类**上游 issue / PR 引用**保持原样——它们指向上游的历史讨论，本 fork 没有对应编号

署名 `(c) ludoux` 在 banner 与生成的 Markdown 页脚中保留（MIT 许可要求保留版权声明），banner 额外标注 `fork maintained by YHuanheg`。

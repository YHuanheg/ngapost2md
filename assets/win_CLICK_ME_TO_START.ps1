# ngapost2md Windows 启动器
# 由同目录下的 win_CLICK_ME_TO_START.bat 调用，也可直接右键"使用 PowerShell 运行"。
# 注意：本文件必须保存为 UTF-8 with BOM，否则 Windows PowerShell 5.1 会把中文读成乱码。

$ErrorActionPreference = 'Continue'

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
Set-Location -LiteralPath $scriptDir

$exe = Join-Path $scriptDir 'ngapost2md.exe'
$cfg = Join-Path $scriptDir 'config.ini'

# 统一按 UTF-8 输出，避免程序日志里的中文乱码
try { chcp 65001 | Out-Null } catch { }
try { [Console]::OutputEncoding = [System.Text.Encoding]::UTF8 } catch { }
try { $Host.UI.RawUI.WindowTitle = 'ngapost2md 启动器' } catch { }

function Stop-Script {
    param([string]$Message)
    Write-Host ''
    Write-Host $Message -ForegroundColor Red
    Write-Host ''
    Read-Host '按回车键退出' | Out-Null
    exit 1
}

if (-not (Test-Path -LiteralPath $exe)) {
    Stop-Script '[错误] 当前目录找不到 ngapost2md.exe，请把本脚本与它放在同一个目录里。'
}

function Test-DirWritable {
    $probe = Join-Path $scriptDir ('.write_probe_' + [Guid]::NewGuid().ToString('N'))
    try {
        [System.IO.File]::WriteAllText($probe, 'x')
        Remove-Item -LiteralPath $probe -Force -ErrorAction SilentlyContinue
        return $true
    }
    catch {
        return $false
    }
}

function Get-ServerPort {
    if (Test-Path -LiteralPath $cfg) {
        $m = Select-String -LiteralPath $cfg -Pattern '^\s*port\s*=\s*(\d+)' | Select-Object -First 1
        if ($m) { return $m.Matches[0].Groups[1].Value }
    }
    return '8080'
}

function Wait-Menu {
    Write-Host ''
    Read-Host '按回车键返回菜单' | Out-Null
}

$dirWritable = Test-DirWritable

while ($true) {
    Clear-Host
    Write-Host '============================================================' -ForegroundColor Cyan
    Write-Host '  ngapost2md 启动器' -ForegroundColor Cyan
    Write-Host '------------------------------------------------------------'
    Write-Host ('  程序目录 : ' + $scriptDir)
    if (Test-Path -LiteralPath $cfg) {
        Write-Host '  配置文件 : config.ini  [已找到]' -ForegroundColor Green
    }
    else {
        Write-Host '  配置文件 : config.ini  [缺失，选 [1]/[3] 时会自动生成]' -ForegroundColor Yellow
    }
    if ($dirWritable) {
        Write-Host '  目录可写 : 是' -ForegroundColor Green
    }
    else {
        Write-Host '  目录可写 : 否（Server 模式/配置保存可能失败）' -ForegroundColor Red
        Write-Host '             建议把整个文件夹移到 D:\ngapost2md 这类位置，或右键以管理员身份运行。' -ForegroundColor Red
    }
    Write-Host '============================================================' -ForegroundColor Cyan
    Write-Host ''
    Write-Host '  [1] 下载 / 增量更新帖子      （命令行模式）'
    Write-Host '  [2] 只导出某位用户的楼层      （命令行模式）'
    Write-Host '  [3] 启动 Server 模式          （WebUI + HTTP API）'
    Write-Host '  [4] 用记事本编辑 config.ini'
    Write-Host '  [5] 查看版本与构建信息'
    Write-Host '  [6] 生成默认 config.ini       （会覆盖，先自动备份）'
    Write-Host '  [7] 检查更新                  （查询上游仓库）'
    Write-Host '  [0] 退出'
    Write-Host ''

    $choice = (Read-Host '请输入序号并回车').Trim()

    switch ($choice) {
        '1' {
            Clear-Host
            Write-Host '=== 下载 / 增量更新帖子 ===' -ForegroundColor Cyan
            Write-Host ''
            Write-Host '可以输入帖子 tid（例如 5935947），也可以直接粘贴 NGA 链接。'
            Write-Host '本地已存在该帖时会自动做增量更新。'
            Write-Host ''
            $tid = (Read-Host '请输入 tid 或链接').Trim()
            if ($tid -eq '') { continue }
            Write-Host ''
            Write-Host '------------------------------------------------------------'
            & $exe $tid
            Write-Host '------------------------------------------------------------'
            Write-Host ('程序已退出（返回码 ' + $LASTEXITCODE + '）。')
            Wait-Menu
        }
        '2' {
            Clear-Host
            Write-Host '=== 只导出某位用户的楼层 ===' -ForegroundColor Cyan
            Write-Host ''
            $tid = (Read-Host '请输入 tid 或链接').Trim()
            if ($tid -eq '') { continue }
            $aid = (Read-Host '请输入要筛选的用户 id（authorid）').Trim()
            if ($aid -eq '') { continue }
            Write-Host ''
            Write-Host '------------------------------------------------------------'
            & $exe $tid --authorid $aid
            Write-Host '------------------------------------------------------------'
            Write-Host ('程序已退出（返回码 ' + $LASTEXITCODE + '）。')
            Wait-Menu
        }
        '3' {
            Clear-Host
            Write-Host '=== 启动 Server 模式 ===' -ForegroundColor Cyan
            Write-Host ''
            $port = Get-ServerPort
            Write-Host ('启动后请在浏览器打开:  http://127.0.0.1:' + $port) -ForegroundColor Green
            Write-Host '用户名固定为 admin，密码见下方日志（首次启动会自动生成并写入 config.ini）。'
            Write-Host '按 Ctrl+C 可以停止服务，停止后回到本菜单。'
            Write-Host ''
            Write-Host '------------------------------------------------------------'
            & $exe serve
            Write-Host '------------------------------------------------------------'
            Write-Host ('服务已停止（返回码 ' + $LASTEXITCODE + '）。')
            Wait-Menu
        }
        '4' {
            if (-not (Test-Path -LiteralPath $cfg)) {
                Clear-Host
                Write-Host '还没有 config.ini，请先选 [6] 生成一份默认配置。' -ForegroundColor Yellow
                Wait-Menu
                continue
            }
            Start-Process -FilePath 'notepad.exe' -ArgumentList ('"' + $cfg + '"')
        }
        '5' {
            Clear-Host
            & $exe -v
            Wait-Menu
        }
        '6' {
            Clear-Host
            Write-Host '=== 生成默认 config.ini ===' -ForegroundColor Cyan
            Write-Host ''
            if (Test-Path -LiteralPath $cfg) {
                Copy-Item -LiteralPath $cfg -Destination ($cfg + '.bak') -Force
                Write-Host '原配置已备份为 config.ini.bak' -ForegroundColor Yellow
            }
            Write-Host ''
            & $exe --gen-config-file
            Write-Host ''
            Write-Host '记得填写 ua / ngaPassportUid / ngaPassportCid 三项，可用菜单 [4] 直接编辑。'
            Wait-Menu
        }
        '7' {
            Clear-Host
            Write-Host '=== 检查更新 ===' -ForegroundColor Cyan
            Write-Host ''
            Write-Host '说明: 本程序查询的是上游仓库 ludoux/ngapost2md 的最新版本。'
            Write-Host '      本分支版本号（2.0.0-fix）与上游不同，被提示“需要更新”属于正常现象。'
            Write-Host ''
            & $exe --update
            Wait-Menu
        }
        '0' { return }
        default {
            Write-Host ''
            Write-Host '无效的序号，请重新输入。' -ForegroundColor Yellow
            Start-Sleep -Milliseconds 900
        }
    }
}

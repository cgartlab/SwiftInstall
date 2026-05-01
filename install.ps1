# SwiftInstall 一键安装脚本
# 用法: irm https://raw.githubusercontent.com/cgartlab/Software_Install_Script/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$RepoOwner = "cgartlab"
$RepoName = "Software_Install_Script"
$BinaryName = "sis.exe"
$InstallDir = "$env:LOCALAPPDATA\SwiftInstall"

function Write-Info {
    param([string]$Message)
    Write-Host "[SwiftInstall] $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SwiftInstall] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[SwiftInstall] $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "[SwiftInstall] $Message" -ForegroundColor Red
}

function Get-LatestReleaseUrl {
    $apiUrl = "https://api.github.com/repos/$RepoOwner/$RepoName/releases/latest"
    Write-Info "正在查询最新版本信息..."

    try {
        $release = Invoke-RestMethod -Uri $apiUrl -Headers @{
            "User-Agent" = "SwiftInstall-Installer"
        } -TimeoutSec 30
    } catch {
        throw "无法获取最新版本信息: $_"
    }

    $tag = $release.tag_name
    Write-Info "最新版本: $tag"

    # 优先查找带 windows/amd64 或 windows 标识的资产
    $asset = $release.assets | Where-Object {
        $_.name -match "windows.*amd64|windows.*x64|sis.*windows|sis.*\.exe" -or
        $_.name -eq "sis.exe"
    } | Select-Object -First 1

    if (-not $asset) {
        # 回退: 查找任何 exe 文件
        $asset = $release.assets | Where-Object { $_.name -match "\.exe$" } | Select-Object -First 1
    }

    if (-not $asset) {
        throw "未在 Release $tag 中找到可下载的二进制文件。请手动从 Releases 页面下载。"
    }

    return $asset.browser_download_url, $asset.name, $tag
}

function Install-SwiftInstall {
    $downloadUrl, $fileName, $version = Get-LatestReleaseUrl

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $outputPath = Join-Path $InstallDir $BinaryName
    $tempPath = "$outputPath.tmp"

    Write-Info "正在下载 $fileName..."
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile $tempPath -UseBasicParsing -TimeoutSec 120
    } catch {
        throw "下载失败: $_"
    }

    # 如果目标文件正在运行，先重命名旧版本
    if (Test-Path $outputPath) {
        $backupPath = "$outputPath.backup"
        Remove-Item -Path $backupPath -ErrorAction SilentlyContinue
        Rename-Item -Path $outputPath -NewName $backupPath -ErrorAction SilentlyContinue
    }

    Move-Item -Path $tempPath -Destination $outputPath -Force
    Write-Success "已安装到: $outputPath"

    # 验证可执行文件
    try {
        $verOutput = & $outputPath version 2>$null
        Write-Success "验证成功: $verOutput"
    } catch {
        Write-Warn "无法验证版本信息，但文件已安装。"
    }

    return $outputPath
}

function Add-ToPath {
    param([string]$TargetDir)

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")

    if ($userPath -split ";" | Where-Object { $_ -ieq $TargetDir }) {
        Write-Info "安装目录已在用户 PATH 中。"
        return
    }

    if ($machinePath -split ";" | Where-Object { $_ -ieq $TargetDir }) {
        Write-Info "安装目录已在系统 PATH 中。"
        return
    }

    Write-Info "正在将 $TargetDir 添加到用户 PATH..."
    $newPath = "$userPath;$TargetDir"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Success "已添加到用户 PATH。"

    # 同时更新当前会话的 PATH
    $env:Path = "$env:Path;$TargetDir"
}

function Test-WingetAvailable {
    $wg = Get-Command winget -ErrorAction SilentlyContinue
    if (-not $wg) {
        Write-Warn "未检测到 winget。SwiftInstall 依赖 winget 作为包管理器。"
        Write-Warn "Windows 11 已预装 winget；Windows 10 用户请从 Microsoft Store 安装 App Installer。"
        return $false
    }
    Write-Info "winget 已就绪: $($wg.Source)"
    return $true
}

# ==================== 主流程 ====================

Write-Info "开始安装 SwiftInstall..."

# 检查 winget
$hasWinget = Test-WingetAvailable

# 下载并安装
$binaryPath = Install-SwiftInstall

# 添加到 PATH
Add-ToPath -TargetDir $InstallDir

Write-Host ""
Write-Success "SwiftInstall 安装完成！"
Write-Host ""
Write-Host "  安装路径: $binaryPath" -ForegroundColor Gray
Write-Host "  使用方式: sis install    # 批量安装" -ForegroundColor Gray
Write-Host "            sis list       # 查看清单" -ForegroundColor Gray
Write-Host "            sis mirror ustc # 切换国内源" -ForegroundColor Gray
Write-Host ""

if (-not $hasWinget) {
    Write-Warn "请先安装 winget，然后重新打开终端使用 sis。"
}

Write-Info "提示: 如果 `sis` 命令无法识别，请重新打开 PowerShell 窗口。"

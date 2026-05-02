# SwiftInstall 一键安装脚本
# 用法（标准）:
#   irm https://raw.githubusercontent.com/cgartlab/Software_Install_Script/main/install.ps1 | iex
#
# 中国用户加速安装（任选其一）:
#   irm https://cdn.jsdelivr.net/gh/cgartlab/Software_Install_Script@main/install.ps1 | iex
#   $env:SIS_MIRROR='ghproxy.com'; irm https://raw.githubusercontent.com/cgartlab/Software_Install_Script/main/install.ps1 | iex
#   irm https://ghproxy.com/https://raw.githubusercontent.com/cgartlab/Software_Install_Script/main/install.ps1 | iex

$savedEAP = $ErrorActionPreference
$ErrorActionPreference = "Stop"
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

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

function Get-LatestReleaseUrl {
    $apiUrl = "https://api.github.com/repos/$RepoOwner/$RepoName/releases/latest"
    Write-Info "正在查询最新版本信息..."

    $release = $null
    try {
        $release = Invoke-RestMethod -Uri $apiUrl -Headers @{
            "User-Agent" = "SwiftInstall-Installer"
        } -TimeoutSec 30
    } catch {
        throw "无法获取最新版本信息: $_"
    }

    $tag = $release.tag_name
    Write-Info "最新版本: $tag"
    $escapedTag = [regex]::Escape($tag)

    # Tier 1: tag + windows + amd64 + .exe (highest priority)
    $asset = $null
    $asset = @($release.assets | Where-Object {
        $_.name -match "${escapedTag}.*windows.*(amd64|x64).*\.exe$"
    } | Select-Object -First 1)

    # Tier 2: tag + windows + any arch + .exe
    if (-not $asset) {
        $asset = @($release.assets | Where-Object {
            $_.name -match "${escapedTag}.*windows.*\.exe$"
        } | Select-Object -First 1)
    }

    # Tier 3: tag + windows + amd64 + .zip
    if (-not $asset) {
        $asset = @($release.assets | Where-Object {
            $_.name -match "${escapedTag}.*windows.*(amd64|x64)\.zip$"
        } | Select-Object -First 1)
    }

    # Tier 4: any windows + amd64 + .zip (untagged fallback)
    if (-not $asset) {
        $asset = @($release.assets | Where-Object {
            $_.name -match "windows.*(amd64|x64).*\.zip$"
        } | Select-Object -First 1)
    }

    # Tier 5: any windows + amd64 + .exe (untagged fallback)
    if (-not $asset) {
        $asset = @($release.assets | Where-Object {
            $_.name -match "windows.*(amd64|x64).*\.exe$"
        } | Select-Object -First 1)
    }

    # Tier 6: broad fallback (original logic)
    if (-not $asset) {
        $asset = @($release.assets | Where-Object {
            $_.name -match "windows.*(amd64|x64)" -or
            $_.name -match "sis.*windows" -or
            $_.name -match "\.exe$"
        } | Select-Object -First 1)
    }

    if (-not $asset) {
        throw "未在 Release $tag 中找到可下载的二进制文件。请手动从 Releases 页面下载。"
    }

    Write-Info "资产: $($asset.name)"
    return $asset.browser_download_url, $asset.name, $tag
}

function Invoke-WithRetry {
    param(
        [scriptblock]$ScriptBlock,
        [int]$MaxRetries = 3,
        [int]$DelaySec = 3
    )
    $lastErr = $null
    for ($i = 0; $i -lt $MaxRetries; $i++) {
        try {
            return & $ScriptBlock
        } catch {
            $lastErr = $_
            if ($i -lt $MaxRetries - 1) {
                Write-Info "下载失败，$( $i + 1 )/$MaxRetries，重试中...（$($_.Exception.Message)）"
                Start-Sleep -Seconds $DelaySec
            }
        }
    }
    throw $lastErr
}

function Install-SwiftInstall {
    $downloadUrl, $fileName, $version = Get-LatestReleaseUrl

    $mirror = $env:SIS_MIRROR
    if ($mirror) {
        $mirror = $mirror.TrimEnd('/')
        if ($mirror -notmatch '^https?://') { $mirror = "https://$mirror" }
        $downloadUrl = "$mirror/$downloadUrl"
        Write-Info "使用镜像加速: $mirror"
    }

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $outputPath = Join-Path $InstallDir $BinaryName
    $tempPath = "$outputPath.${PID}.tmp"

    $isZip = $fileName -match '\.zip$'

    Write-Info "正在下载 $fileName..."
    try {
        if ($isZip) {
            $zipPath = Join-Path $env:TEMP "sis_dl_$([System.IO.Path]::GetRandomFileName()).zip"
            Invoke-WithRetry -ScriptBlock {
                Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing -TimeoutSec 120
            }
            Write-Info "正在解压 $fileName..."
            $extractDir = Join-Path $env:TEMP "sis_ex_$([System.IO.Path]::GetRandomFileName())"
            try {
                Expand-Archive -Path $zipPath -DestinationPath $extractDir -Force
                $extracted = Get-ChildItem -Path $extractDir -Recurse -Filter "sis*.exe" | Select-Object -First 1
                if (-not $extracted) {
                    $allExes = Get-ChildItem -Path $extractDir -Recurse -Filter "*.exe"
                    if ($allExes) {
                        $extracted = $allExes | Sort-Object Length -Descending | Select-Object -First 1
                    }
                }
                if (-not $extracted) {
                    throw "在 $fileName 中未找到可执行文件"
                }
                Write-Info "提取文件: $($extracted.Name)"
                if (Test-Path $outputPath) {
                    $backupPath = "$outputPath.backup"
                    Remove-Item -Path $backupPath -Force -ErrorAction SilentlyContinue
                    Move-Item -Path $outputPath -Destination $backupPath -Force -ErrorAction SilentlyContinue
                }
                Move-Item -Path $extracted.FullName -Destination $outputPath -Force
            } finally {
                Remove-Item -Path $zipPath -Force -ErrorAction SilentlyContinue
                Remove-Item -Path $extractDir -Recurse -Force -ErrorAction SilentlyContinue
            }
        } else {
            Invoke-WithRetry -ScriptBlock {
                Invoke-WebRequest -Uri $downloadUrl -OutFile $tempPath -UseBasicParsing -TimeoutSec 120
            }
            if (Test-Path $outputPath) {
                $backupPath = "$outputPath.backup"
                Remove-Item -Path $backupPath -Force -ErrorAction SilentlyContinue
                Move-Item -Path $outputPath -Destination $backupPath -Force -ErrorAction SilentlyContinue
            }
            Move-Item -Path $tempPath -Destination $outputPath -Force
        }
    } catch {
        Remove-Item -Path $tempPath -Force -ErrorAction SilentlyContinue
        $errMsg = "安装失败: $_"
        if (Get-Process -Name "sis" -ErrorAction SilentlyContinue) {
            $errMsg += " (请先关闭正在运行的 sis 进程后重试)"
        }
        throw $errMsg
    }

    Write-Success "已安装到: $outputPath"

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

    $pathEntries = @($userPath -split ";" | Where-Object { $_ -ne "" })
    $found = $false
    foreach ($p in $pathEntries) {
        if ($p -and ($p.TrimEnd('\') -eq $TargetDir.TrimEnd('\') -or $p -eq $TargetDir)) {
            $found = $true
            break
        }
    }
    if ($found) {
        Write-Info "安装目录已在用户 PATH 中。"
        return
    }

    $machineEntries = @($machinePath -split ";" | Where-Object { $_ -ne "" })
    foreach ($p in $machineEntries) {
        if ($p -and ($p.TrimEnd('\') -eq $TargetDir.TrimEnd('\') -or $p -eq $TargetDir)) {
            Write-Info "安装目录已在系统 PATH 中。"
            return
        }
    }

    Write-Info "正在将 $TargetDir 添加到用户 PATH..."
    if ($userPath) {
        $newPath = "$userPath;$TargetDir"
    } else {
        $newPath = $TargetDir
    }
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = "$env:Path;$TargetDir"
    Write-Success "已添加到用户 PATH。"
}

function Test-WingetAvailable {
    $wg = Get-Command winget -ErrorAction SilentlyContinue -CommandType Application
    if (-not $wg) {
        Write-Warn "未检测到 winget。SwiftInstall 依赖 winget 作为包管理器。"
        Write-Warn "Windows 11 已预装 winget；Windows 10 用户请从 Microsoft Store 安装 App Installer。"
        return $false
    }
    Write-Info "winget 已就绪: $($wg.Source)"
    return $true
}

try {
    Write-Info "开始安装 SwiftInstall..."

    $hasWinget = Test-WingetAvailable

    $binaryPath = Install-SwiftInstall

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

    if ($env:SIS_MIRROR) {
        $mirrorDisp = $env:SIS_MIRROR.TrimEnd('/')
        if ($mirrorDisp -notmatch '^https?://') { $mirrorDisp = "https://$mirrorDisp" }
        Write-Info "使用了镜像: $mirrorDisp"
    }
} finally {
    $ErrorActionPreference = $savedEAP
}
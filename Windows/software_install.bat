@echo off
setlocal EnableDelayedExpansion

REM License
REM 本项目受 GNU General Public License v3.0 约束
REM 详见 LICENSE 文件或 https://www.gnu.org/licenses/gpl-3.0.html

set "SIS_EXIT_CODE=0"

REM 切到 UTF-8 代码页，保证下方中文提示在常见环境下可读。
REM 清单里的 "- id:" 与结构键（settings/packages/proxy）均为 ASCII，因此 findstr
REM 的匹配不受代码页影响（UTF-8 BOM 只会前缀到第 1 行的 # 注释，不影响提取）；
REM 但清单需保存为 ANSI 或 UTF-8，UTF-16 编码会让 findstr 读不到任何行。
chcp 65001 >nul

REM 检查是否存在软件列表文件
if not exist "software_list.txt" (
    echo Software list file does not exist! Please create the software list file and run the script again.
    exit /b
)

REM 检查是否为管理员权限
net session >nul 2>&1
if %errorlevel% equ 0 (
    REM 询问是否切换到中科大镜像源（中国大陆用户加速）
    set /p USE_MIRROR="是否切换到 USTC 镜像源加速下载？(Y/N, 默认 Y): "
    if /i "!USE_MIRROR!"=="Y" set USE_MIRROR=y
    if /i "!USE_MIRROR!"=="" set USE_MIRROR=y
    if /i "!USE_MIRROR!"=="y" (
        echo 正在切换到 USTC 镜像源...
        winget source remove winget >nul 2>&1
        winget source add winget https://mirrors.ustc.edu.cn/winget-source >nul 2>&1
        if !errorlevel! equ 0 (
            echo 镜像源切换成功！
        ) else (
            echo 镜像源切换失败，将使用官方源。
        )
    )
)

REM 逐行读取软件列表文件并安装软件
REM 兼容两种清单：YAML manifest（缩进的 "- id: xxx" 行）与纯 TXT 列表（每行一个 ID）
REM 说明：本脚本假定清单值是 winget 包 id（与 internal/backend/winget.go 一致），
REM       因此对 "- id:" 提取路径用 --id --exact；对纯 TXT 兜底路径加 --exact --silent。
set "YAML_IDS=%TEMP%\sis_yaml_ids_%RANDOM%%RANDOM%.txt"

REM 只提取“缩进的 - id:”行，避免误匹配 proxy: / category: 等含 "id:" 子串的结构行
findstr /r /c:"^[ 	]*- id:" "software_list.txt" > "%YAML_IDS%" 2>nul

REM winget 安装参数：--id --exact --silent + 接受协议，防模糊匹配与交互挂起
set "WG_FLAGS=--id --exact --silent --accept-package-agreements --accept-source-agreements"
set /a OK_COUNT=0
set /a FAIL_COUNT=0

REM findstr 找到 "- id:" 行时 errorlevel 为 0 → YAML 模式；否则退回纯 TXT 模式
if !errorlevel! equ 0 (
    for /f "usebackq tokens=1,* delims=:" %%a in ("%YAML_IDS%") do (
        set "PKG=%%b"
        REM 去掉 ID 前的空格
        for /f "tokens=*" %%c in ("!PKG!") do set "PKG=%%c"
        if not "!PKG!"=="" (
            echo Installing software: !PKG!
            winget install "!PKG!" !WG_FLAGS!
            if !errorlevel! equ 0 (
                set /a OK_COUNT+=1
            ) else (
                echo Failed to install: !PKG!
                set /a FAIL_COUNT+=1
            )
        )
    )
) else (
    REM 纯 TXT 兜底：除跳过 # 注释外，还要跳过 YAML 结构行
    REM （settings:/packages:/proxy: 等顶层键，以及 category:/kind:/skip_existing: 等缩进属性行），
    REM 以防清单被当纯 TXT 处理时把结构行误当包名。
    REM 判据：winget 包 id 不含冒号，而所有 YAML 结构行/属性行都形如 "key: value" 含冒号，
    REM 故含冒号的行一律跳过——这样纯 TXT 列表（每行一个 id，无冒号）仍照常安装。
    REM 兜底同样加 --exact --silent 防模糊匹配与交互挂起。
    for /f "usebackq tokens=*" %%a in ("software_list.txt") do (
        set "PKG=%%a"
        if not "!PKG!"=="" if not "!PKG:~0,1!"=="#" (
            if "!PKG!"=="!PKG::=!" (
                echo Installing software: !PKG!
                winget install "!PKG!" --exact --silent --accept-package-agreements --accept-source-agreements
                if !errorlevel! equ 0 (
                    set /a OK_COUNT+=1
                ) else (
                    echo Failed to install: !PKG!
                    set /a FAIL_COUNT+=1
                )
            )
        )
    )
)
del "%YAML_IDS%" >nul 2>&1

REM 如实汇报结果（不再无条件打印成功语）
echo.
echo Install finished: !OK_COUNT! succeeded, !FAIL_COUNT! failed.
if !FAIL_COUNT! gtr 0 set "SIS_EXIT_CODE=1"

REM 如果切换过镜像源，询问是否恢复
if /i "!USE_MIRROR!"=="y" (
    set /p RESET="是否恢复为官方源？(Y/N, 默认 Y): "
    if /i "!RESET!"=="Y" set RESET=y
    if /i "!RESET!"=="" set RESET=y
    if /i "!RESET!"=="y" (
        winget source reset --force >nul 2>&1
        echo 已恢复官方源。
    )
)

pause
exit /b !SIS_EXIT_CODE!

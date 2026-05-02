@echo off
setlocal EnableDelayedExpansion

REM License
REM 本项目受 GNU General Public License v3.0 约束
REM 详见 LICENSE 文件或 https://www.gnu.org/licenses/gpl-3.0.html

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
for /f "tokens=*" %%a in (software_list.txt) do (
    echo Installing software: %%a
    winget install %%a 
)

echo All software is already installed!

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

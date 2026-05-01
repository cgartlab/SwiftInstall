@ECHO off

REM License
REM 本项目受 GNU General Public License v3.0 约束
REM 详见 LICENSE 文件或 https://www.gnu.org/licenses/gpl-3.0.html

REM 检查是否存在软件列表文件
if not exist "software_list.txt" (
    echo Software list file does not exist! Please create the software list file and run the script again.
    exit /b
)

REM 判断v2rayN是否运行
tasklist | find /i "v2rayN.exe" >nul
if %errorlevel% EQU 0 (
    ECHO Found v2rayN running. Continuing.
    cls
    REM 逐行读取软件列表文件并安装软件，且使用http代理进行加速
    for /f "tokens=*" %%a in (software_list.txt) do (
        ECHO 正在安装: %%a
        winget install %%a --proxy http://127.0.0.1:10809
    )
) else (
    ECHO Error: v2rayN is not running, please run this script after v2rayN is running.
    pause > nul
    exit
)

ECHO All software is already installed!
pause

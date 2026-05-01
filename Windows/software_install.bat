@echo off

REM License
REM 本项目受 GNU General Public License v3.0 约束
REM 详见 LICENSE 文件或 https://www.gnu.org/licenses/gpl-3.0.html

REM 检查是否存在软件列表文件
if not exist "software_list.txt" (
    echo Software list file does not exist! Please create the software list file and run the script again.
    exit /b
)

REM 逐行读取软件列表文件并安装软件
for /f "tokens=*" %%a in (software_list.txt) do (
    echo Installing software: %%a
    winget install %%a 
)

echo All software is already installed!
pause

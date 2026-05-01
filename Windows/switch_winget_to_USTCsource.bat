@Echo off&setlocal,EnableDelayedExpansion

REM License
REM ����Ŀ�� GNU General Public License v3.0 Լ��
REM ���� LICENSE �ļ��� https://www.gnu.org/licenses/gpl-3.0.html

REM ����Ƿ��Թ���ԱȨ������
net session >nul 2>&1
if %errorlevel% neq 0 (
    cls
    title   ��ʼ��ʧ�ܣ���Ҫ����ԱȨ��......
    echo:
    echo:
    echo:
    echo:
    echo:       ______________________________________________________________
    echo:
    echo:             ��ʼ��ʧ�ܣ�
    echo:
    echo:             ����: ��Ҫ����ԱȨ�ޣ�
    echo:             ������������Ҽ������˽ű���ѡ�� "�Թ���Ա��������"
    echo:             __________________________________________________  
    echo:
    echo:             �޷�����ִ�У��밴������˳���
    echo:
    echo:       ______________________________________________________________
    echo:
    pause > nul
    taskkill /f /im cmd.exe 
    exit /b )

:Choice
cls
title   Software Install Script
echo:
echo:
echo:
echo:
echo:       ______________________________________________________________
echo:
echo:             �л� Winget �б�Դ
echo:
echo:             ���ű�����������Ϊ:
echo:             1.���� Winget �б�ԴΪ�й���ѧ������ѧ����Դ
echo:             2.���� Winget ��������
echo:
echo:             __________________________________________________      
echo:
echo:             [Y] ͬ�� 
echo:             [N] �ܾ�
echo:             [R] �����б�Դ
echo:
echo:       ______________________________________________________________
echo:
set /p Choice=������(Y/N/R)�Լ���:
IF /i "!Choice!"=="Y" Goto :Next
IF /i "!Choice!"=="R" Goto :Rollback
IF /i "!Choice!"=="N" Goto :End
Echo ������Y/N/R���밴��������ء�
Pause>Nul&Goto :Choice

REM ͬ��ִ�п�
:Next
echo ���ڽ� Winget �б�Դ�л����й���ѧ������ѧ����Դ
winget source remove winget
winget source add winget https://mirrors.ustc.edu.cn/winget-source

echo ���ڼ��� Winget ��������
winget settings --enable ProxyCommandLineOptions
taskkill /f /im cmd.exe
exit /b

REM ����ִ�п�
:Rollback
echo ���� Winget �б�Դ
winget source reset --force
taskkill /f /im cmd.exe
exit /b

REM ��ִֹ�п�
:End
ECHO �û�ȡ����װ�������˳���
taskkill /f /im cmd.exe
exit /b

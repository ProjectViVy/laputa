@echo off
REM Laputa Supervisor — keep laputa.exe :7373 alive.
REM Usage:
REM   laputa-supervisor.cmd [loop]
REM Exit codes: 0=ok (running or spawned), 1=laputa.exe missing, 2=spawn failed.

setlocal
set "EXE=C:\Users\Administrator\Desktop\projects\laputa\laputa.exe"
set "PORT=7373"
set "DIR=C:\Users\Administrator\.laputa"
set "LOG=%DIR%\supervisor.log"
set "LOCKDIR=%DIR%\sections"

if not exist "%EXE%" (
    echo FATAL: %EXE% missing >> "%LOG%"
    exit /b 1
)

REM Probe port (skip spawn if already LISTENING)
netstat -ano | findstr ":%PORT% " | findstr "LISTENING" >nul
if not errorlevel 1 (
    echo [%date% %time%] supervisor: laputa already LISTENING on :%PORT%, skipping >> "%LOG%"
    if /I "%1"=="loop" goto :loop
    exit /b 0
)

REM Clean stale lock files from prior crashes
if exist "%LOCKDIR%" del /Q "%LOCKDIR%\*.lock" 2>nul

REM Spawn detached
echo [%date% %time%] supervisor: spawning laputa.exe :%PORT% >> "%LOG%"
start "" /B "%EXE%" -cmd serve -serve-addr 127.0.0.1:%PORT% -store file -dir "%DIR%"

REM Verify (1s grace)
timeout /t 1 /nobreak >nul 2>&1
netstat -ano | findstr ":%PORT% " | findstr "LISTENING" >nul
if errorlevel 1 (
    echo [%date% %time%] supervisor: spawn failed >> "%LOG%"
    exit /b 2
)
echo [%date% %time%] supervisor: spawn OK >> "%LOG%"

if /I "%1"=="loop" goto :loop
exit /b 0

:loop
echo [%date% %time%] entering supervisor loop >> "%LOG%"
:loop_check
timeout /t 5 /nobreak >nul 2>&1
netstat -ano | findstr ":%PORT% " | findstr "LISTENING" >nul
if errorlevel 1 (
    echo [%date% %time%] supervisor: laputa died, respawning >> "%LOG%"
    call "%~f0"
)
goto :loop_check
@echo off
REM Setup nightly auto-iteration scheduled task
REM Run as Administrator

schtasks /Delete /TN "AgentSandbox_Nightly" /F 2>NUL

schtasks /Create ^
    /TN "AgentSandbox_Nightly" ^
    /TR "\"C:\Program Files\Git\bin\bash.exe\" -l -c \"/c/Users/Administrator/ai-sandbox/automation/nightly.sh\"" ^
    /SC DAILY ^
    /ST 00:00 ^
    /RL HIGHEST ^
    /F

if %ERRORLEVEL% EQU 0 (
    echo.
    echo [OK] Nightly task created!
    echo Schedule: Every day at 00:00
    echo.
    schtasks /Query /TN "AgentSandbox_Nightly"
) else (
    echo [ERROR] Failed. Run as Administrator.
)
pause

@echo off
REM ============================================================================
REM Create Windows Scheduled Task to run AgentSandbox overnight build
REM Run this script once as Administrator to set up the schedule
REM ============================================================================

REM Delete existing task if any
schtasks /Delete /TN "AgentSandbox_OvernightBuild" /F 2>NUL

REM Create scheduled task for tonight at midnight
REM Uses Git Bash to run the orchestration script
schtasks /Create ^
    /TN "AgentSandbox_OvernightBuild" ^
    /TR "\"C:\Program Files\Git\bin\bash.exe\" -l -c \"/c/Users/Administrator/ai-sandbox/automation/run-overnight.sh\"" ^
    /SC ONCE ^
    /ST 00:00 ^
    /SD %date:~0,10% ^
    /RL HIGHEST ^
    /F

if %ERRORLEVEL% EQU 0 (
    echo.
    echo [SUCCESS] Scheduled task created!
    echo Task Name: AgentSandbox_OvernightBuild
    echo Schedule: Tonight at 00:00
    echo Script: /c/Users/Administrator/ai-sandbox/automation/run-overnight.sh
    echo.
    echo To check status: schtasks /Query /TN "AgentSandbox_OvernightBuild"
    echo To run now:      schtasks /Run /TN "AgentSandbox_OvernightBuild"
    echo To delete:       schtasks /Delete /TN "AgentSandbox_OvernightBuild" /F
) else (
    echo [ERROR] Failed to create scheduled task. Run as Administrator.
)

pause

@echo off
setlocal enabledelayedexpansion

echo DevProxy Service Installer
echo ==========================
echo.

REM Check if running as administrator
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ERROR: This script must be run as Administrator!
    echo Please right-click and select "Run as administrator"
    pause
    exit /b 1
)

REM Get the directory where this script is located
set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%.."

REM Build the absolute path to the executable
pushd "%PROJECT_ROOT%"
set "DEVPROXY_PATH=%CD%\devproxy.exe"
popd

REM Check if devproxy.exe exists
if not exist "%DEVPROXY_PATH%" (
    echo ERROR: devproxy.exe not found at %DEVPROXY_PATH%
    echo Please build the project first using: go build -o devproxy.exe cmd/devproxy/main.go
    pause
    exit /b 1
)

REM Parse command line arguments
set "ACTION=%1"
if "%ACTION%"=="" set "ACTION=install"

if /i "%ACTION%"=="install" goto :install
if /i "%ACTION%"=="uninstall" goto :uninstall
if /i "%ACTION%"=="start" goto :start
if /i "%ACTION%"=="stop" goto :stop
if /i "%ACTION%"=="status" goto :status
goto :usage

:install
echo Installing DevProxy service...
sc create DevProxy binPath= "\"%DEVPROXY_PATH%\"" start= auto DisplayName= "DevProxy - Local Admin API"
if %errorLevel% equ 0 (
    echo Service installed successfully!
    echo.
    echo Starting service...
    sc start DevProxy
    if %errorLevel% equ 0 (
        echo Service started successfully!
    ) else (
        echo Failed to start service. You can start it manually with: sc start DevProxy
    )
) else (
    echo Failed to install service.
)
goto :end

:uninstall
echo Stopping DevProxy service...
sc stop DevProxy >nul 2>&1
echo Uninstalling DevProxy service...
sc delete DevProxy
if %errorLevel% equ 0 (
    echo Service uninstalled successfully!
) else (
    echo Failed to uninstall service.
)
goto :end

:start
echo Starting DevProxy service...
sc start DevProxy
if %errorLevel% equ 0 (
    echo Service started successfully!
) else (
    echo Failed to start service.
)
goto :end

:stop
echo Stopping DevProxy service...
sc stop DevProxy
if %errorLevel% equ 0 (
    echo Service stopped successfully!
) else (
    echo Failed to stop service.
)
goto :end

:status
echo DevProxy Service Status:
echo ========================
sc query DevProxy
goto :end

:usage
echo Usage: service.bat [install^|uninstall^|start^|stop^|status]
echo.
echo   install   - Install and start the DevProxy service
echo   uninstall - Stop and uninstall the DevProxy service
echo   start     - Start the DevProxy service
echo   stop      - Stop the DevProxy service
echo   status    - Show the current status of the DevProxy service
echo.
echo Default action is 'install' if no argument is provided.
goto :end

:end
pause
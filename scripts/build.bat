@echo off
echo Building DevProxy...
echo.

REM Set build environment for Windows
set GOOS=windows
set GOARCH=amd64

REM Build the main DevProxy executable
echo Building devproxy.exe...
go build -o devproxy.exe cmd/devproxy/main.go
if %errorlevel% neq 0 (
    echo Failed to build devproxy.exe
    pause
    exit /b %errorlevel%
)

REM Build the devctl client tool
echo Building devctl.exe...
go build -o devctl.exe cmd/devctl/main.go
if %errorlevel% neq 0 (
    echo Failed to build devctl.exe
    pause
    exit /b %errorlevel%
)

REM Build the tray application
echo Building devproxy-tray.exe...
go build -ldflags="-H windowsgui" -o devproxy-tray.exe cmd/devproxy-tray/main.go cmd/devproxy-tray/icon.go
if %errorlevel% neq 0 (
    echo Warning: Failed to build devproxy-tray.exe (GUI dependencies may be missing)
    echo You can still use DevProxy without the tray application
)

echo.
echo Build completed successfully!
echo.
echo Files created:
echo - devproxy.exe (main service)
echo - devctl.exe (CLI client)
if exist devproxy-tray.exe (
    echo - devproxy-tray.exe (system tray GUI)
)
echo.
echo To install as a Windows service, run:
echo   scripts\service.bat install
echo.
echo To run the system tray GUI:
echo   devproxy-tray.exe
echo.
pause
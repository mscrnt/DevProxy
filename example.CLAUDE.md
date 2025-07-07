# Working with Windows from WSL via DevProxy

## Overview
When working in WSL, you can execute Windows commands using DevProxy - a secure local admin API that runs on the Windows host. This allows you to interact with Windows-native tools and processes that aren't accessible from WSL.

## Key Information for Claude

### How to Use DevProxy from WSL
Since you're in WSL and DevProxy only accepts localhost connections, you need to use the `devctl.exe` executable directly:

```bash
/path/to/DevProxy/devctl.exe -token YOUR_TOKEN_HERE -cwd C:\\Path\\To\\Project command arguments
```

### Authentication
**Token**: `YOUR_TOKEN_HERE`

This token must be included in all devctl commands.

### Important Paths
- DevProxy location: `/path/to/DevProxy/devctl.exe`
- When specifying paths, use Windows format: `C:\\Path\\To\\Folder`
- Current project path in Windows: `D:\\Projects\\AgentM` (or wherever you're working)

## Common Tasks

### 1. Running PowerShell Commands
```bash
/path/to/DevProxy/devctl.exe -token YOUR_TOKEN_HERE -cwd D:\\Projects\\CurrentProject powershell -Command "Your-Command-Here"
```

### 2. Running Python Scripts on Windows
```bash
/path/to/DevProxy/devctl.exe -token YOUR_TOKEN_HERE -cwd D:\\Projects\\CurrentProject python script.py
```

### 3. Managing Windows Processes
To stop a process:
```bash
/path/to/DevProxy/devctl.exe -token YOUR_TOKEN_HERE -cwd D:\\Projects\\CurrentProject powershell -Command "Get-Process ProcessName | Stop-Process -Force"
```

### 4. Running Batch Files or PowerShell Scripts
```bash
/path/to/DevProxy/devctl.exe -token YOUR_TOKEN_HERE -cwd D:\\Projects\\CurrentProject powershell .\\script.ps1
```

## Working with WSL vs Windows Paths

### Path Translation
- WSL path: `/mnt/d/Projects/MyProject/file.txt`
- Windows path: `D:\\Projects\\MyProject\\file.txt`

### When to Use Which Environment
- **Use WSL Bash**: For Linux tools, grep, sed, curl to localhost services
- **Use DevProxy**: For Windows-specific tools, PowerShell, managing Windows services, accessing Windows-only programs

## Limitations and Workarounds

### 1. Command Parsing Issues
DevProxy may have issues with complex PowerShell pipelines. If you encounter parsing errors:
- Create a `.ps1` script file with the complex command
- Execute the script file instead

### 2. Long-Running Commands
Commands that run indefinitely (like starting a server) will timeout. For these:
- Create scripts that start processes in the background
- Use Windows Task Scheduler or services for persistent processes

### 3. Output Limitations
- Large outputs may be truncated
- Interactive commands won't work (no stdin)

## Security Notes
- DevProxy only accepts these whitelisted commands: `go`, `msbuild`, `signtool`, `powershell`, `dotnet`, `gcc`, `g++`, `make`, `cmake`, `npm`, `node`, `python`, `pip`
- System commands like `reg`, `shutdown`, `format`, etc. are blocked
- Only allowed paths can be accessed (typically user project directories)

## Examples for Common Development Tasks

### Check if a Windows service is running
```bash
/path/to/DevProxy/devctl.exe -token YOUR_TOKEN_HERE -cwd C:\\Windows\\System32 powershell -Command "Get-Service ServiceName"
```

### Install npm packages on Windows
```bash
/path/to/DevProxy/devctl.exe -token YOUR_TOKEN_HERE -cwd D:\\Projects\\MyProject npm install
```

### Build a .NET project
```bash
/path/to/DevProxy/devctl.exe -token YOUR_TOKEN_HERE -cwd D:\\Projects\\MyProject dotnet build
```

## Remember
- Always use absolute Windows paths with double backslashes
- The token is required for every command
- Create script files for complex multi-line operations
- DevProxy provides a bridge between WSL and Windows - use it when you need Windows-specific functionality
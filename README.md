# DevProxy

A local-only admin bridge for secure LLM development workflows

## ⚠️ IMPORTANT SECURITY WARNING ⚠️

**DevProxy provides elevated system access through an API. USE AT YOUR OWN RISK.**

This tool allows external programs (including AI assistants) to execute commands on your Windows system with elevated privileges. While security measures are in place, you should understand the risks:

- **Elevated Privileges**: Commands run with the same permissions as the DevProxy service
- **Potential for Misuse**: Even with whitelisting, PowerShell access can be dangerous
- **Token Security**: Anyone with the API token can execute commands
- **AI Integration Risks**: AI assistants may execute unintended commands

**By using DevProxy, you acknowledge these risks and take full responsibility for any consequences.**

## Overview

DevProxy is a secure, local-only admin API for Windows development tasks. It runs as an elevated Windows service and exposes a localhost-only JSON API that allows trusted local agents (like Claude running in WSL) to execute safe build-related commands and file operations.

## Features

- 🔒 **Secure by Design**: Localhost-only binding (configurable port)
- 🛡️ **Token Authentication**: API token required for all requests
- ✅ **Command Whitelisting**: Only allows pre-approved development commands
- 📁 **Path Restrictions**: Operations limited to approved directories
- 🚫 **System Protection**: Blocks access to Windows system directories
- 📝 **Comprehensive Logging**: All requests logged with full details
- 🪟 **Windows Service**: Runs as an auto-start Windows service
- 🖥️ **System Tray GUI**: Admin panel for easy configuration

## Installation

1. **Build the executables:**
   ```batch
   scripts\build.bat
   ```
   Or manually:
   ```bash
   go build -o devproxy.exe cmd/devproxy/main.go
   go build -o devctl.exe cmd/devctl/main.go
   ```

2. **Run interactively (for testing):**
   ```batch
   devproxy.exe
   ```

3. **Install as Windows service (Run as Administrator):**
   ```batch
   scripts\service.bat install
   ```

## Critical Setup Steps

### 1. Generate and Secure Your Token

When DevProxy runs for the first time, it generates a unique API token. This token is your only line of defense against unauthorized access.

1. Run DevProxy interactively first: `devproxy.exe`
2. Copy the generated token from the console output
3. Find the token in `config/config.json` under the `"token"` field
4. **NEVER** commit this token to version control
5. **NEVER** share this token publicly

### 2. Configure for AI Assistant Use

If you're using DevProxy with an AI assistant (Claude, ChatGPT, etc.):

1. Create a `CLAUDE.md` or similar file in your project
2. Add the token to this file for the AI to use
3. **IMPORTANT**: Add this file to `.gitignore`
4. Consider the risks of giving an AI system access

Example CLAUDE.md format:
```markdown
# DevProxy Access

**Token**: your-generated-token-here

Use this token with devctl.exe or API calls to execute Windows commands.
```

### 3. Review and Restrict Allowed Commands

The default configuration includes PowerShell, which is powerful but dangerous. Review `config/config.json` and remove any commands you don't need:

```json
{
  "allowed_commands": [
    "go", "msbuild", "dotnet", "npm", "node", "python", "pip"
    // Consider removing "powershell" if not needed
  ]
}
```

## Configuration

On first run, DevProxy creates a `config/config.json` file with:
- Generated API token (save this securely!)
- Allowed commands list
- Allowed path patterns
- Log file location
- Port number (default: 2223)

Example configuration:
```json
{
  "api_token": "your-generated-token-here",
  "allowed_commands": [
    "go", "msbuild", "signtool", "powershell",
    "dotnet", "gcc", "g++", "make", "cmake",
    "npm", "node", "python", "pip"
  ],
  "allowed_paths": [
    "C:\\Dev",
    "C:\\Users\\*\\Projects",
    "C:\\Users\\*\\source\\repos"
  ],
  "log_file": "logs\\log.txt",
  "port": 2223
}
```

⚠️ **Path Wildcard Warning**: Wildcards in paths (e.g., `C:\Users\*\Projects`) may not work as expected. Use specific paths when possible.

### System Tray GUI

Run `devproxy-tray.exe` to access the admin panel where you can:
- Start/Stop/Restart the service
- Change the port number
- Manage allowed paths
- View and regenerate the API token
- Edit allowed commands

## API Usage

### Run Command Endpoint

**POST** `/run`

Headers:
- `X-Admin-Token: your-api-token`
- `Content-Type: application/json`

Request body:
```json
{
  "command": "go",
  "args": ["build", "-o", "out.exe"],
  "cwd": "C:\\Dev\\MyApp"
}
```

Response:
```json
{
  "stdout": "...",
  "stderr": "",
  "exit_code": 0
}
```

## Security Features

### Blocked Operations
- System directories (C:\Windows, C:\Program Files, etc.)
- Registry modifications
- Service management commands
- System shutdown/restart commands
- Path traversal attempts (..)

### Blocked Keywords
- `reg`, `shutdown`, `format`, `schtasks`, `sc`, `net`, `bcdedit`, `diskpart`

## Using with AI Assistants

### From WSL (e.g., Claude)

AI assistants running in WSL can use devctl.exe directly:

```bash
/mnt/c/path/to/devctl.exe -token YOUR_TOKEN -cwd C:\\Dev command args
```

### Security Considerations for AI Use

1. **Audit Regularly**: Review `logs/log.txt` frequently
2. **Limit Scope**: Only allow paths where AI should operate
3. **Remove Dangerous Commands**: Consider removing PowerShell access
4. **Monitor Activity**: Watch for unexpected command patterns
5. **Revoke Access**: Regenerate token if you suspect misuse

## Service Management

```batch
# Install and start service
scripts\service.bat install

# Stop service
scripts\service.bat stop

# Start service
scripts\service.bat start

# Check service status
scripts\service.bat status

# Uninstall service
scripts\service.bat uninstall
```

## Testing from WSL

Using curl:
```bash
curl -X POST http://127.0.0.1:2223/run \
  -H "X-Admin-Token: your-token-here" \
  -H "Content-Type: application/json" \
  -d '{
    "command": "go",
    "args": ["version"],
    "cwd": "C:\\Dev"
  }'
```

## Logs

All operations are logged to `logs/log.txt` in JSON format, including:
- Timestamp
- Source IP
- Command and arguments
- Working directory
- Output (stdout/stderr)
- Exit code
- Status (completed/rejected)
- Rejection reason (if applicable)

**Review logs regularly to ensure no unauthorized or unintended commands are being executed.**

## Best Practices

1. **Token Management**
   - Regenerate tokens periodically
   - Never share tokens in public repositories
   - Use different tokens for different projects

2. **Command Restrictions**
   - Remove commands you don't need
   - Avoid PowerShell if possible
   - Use specific tools instead of general interpreters

3. **Path Restrictions**
   - Be as specific as possible with allowed paths
   - Avoid wildcards when possible
   - Never allow access to system directories

4. **Monitoring**
   - Check logs daily
   - Set up alerts for suspicious commands
   - Review AI assistant interactions

## Disclaimer

DevProxy is provided "as is" without warranty of any kind. The authors are not responsible for any damage or data loss resulting from the use of this software. By using DevProxy, you acknowledge that you understand the security implications and accept all risks.

## License

MIT License

## Author

Kenneth Blossom
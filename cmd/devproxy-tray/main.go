//go:build windows
// +build windows

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/getlantern/systray"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type Config struct {
	APIToken     string   `json:"api_token"`
	AllowedCmds  []string `json:"allowed_commands"`
	AllowedPaths []string `json:"allowed_paths"`
	LogFile      string   `json:"log_file"`
	Port         int      `json:"port"`
}

var (
	config       *Config
	configPath   string
	mainWindow   *walk.MainWindow
	portEdit     *walk.NumberEdit
	pathsEdit    *walk.TextEdit
	tokenEdit    *walk.LineEdit
	statusLabel  *walk.Label
)

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(getIcon())
	systray.SetTitle("DevProxy")
	systray.SetTooltip("DevProxy Admin Panel")

	mShow := systray.AddMenuItem("Show Admin Panel", "Configure DevProxy settings")
	systray.AddSeparator()
	
	mStart := systray.AddMenuItem("Start Service", "Start DevProxy service")
	mStop := systray.AddMenuItem("Stop Service", "Stop DevProxy service")
	mRestart := systray.AddMenuItem("Restart Service", "Restart DevProxy service")
	
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit DevProxy Tray")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				showAdminPanel()
			case <-mStart.ClickedCh:
				controlService("start")
			case <-mStop.ClickedCh:
				controlService("stop")
			case <-mRestart.ClickedCh:
				controlService("restart")
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onExit() {
}

func showAdminPanel() {
	loadConfig()
	
	var dlg *walk.Dialog
	var acceptPB, cancelPB *walk.PushButton

	_, err := Dialog{
		AssignTo:      &dlg,
		Title:         "DevProxy Admin Panel",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		MinSize:       Size{600, 500},
		Layout:        VBox{},
		Children: []Widget{
			Composite{
				Layout: VBox{},
				Children: []Widget{
					GroupBox{
						Title:  "Service Control",
						Layout: HBox{},
						Children: []Widget{
							PushButton{
								Text: "Start",
								OnClicked: func() {
									controlService("start")
									updateStatus()
								},
							},
							PushButton{
								Text: "Stop",
								OnClicked: func() {
									controlService("stop")
									updateStatus()
								},
							},
							PushButton{
								Text: "Restart",
								OnClicked: func() {
									controlService("restart")
									updateStatus()
								},
							},
							Label{
								AssignTo: &statusLabel,
								Text:     "Status: Unknown",
							},
						},
					},
					GroupBox{
						Title:  "Network Settings",
						Layout: Grid{Columns: 2},
						Children: []Widget{
							Label{Text: "Port:"},
							NumberEdit{
								AssignTo: &portEdit,
								Value:    float64(getPort()),
								MinValue: 1024,
								MaxValue: 65535,
							},
						},
					},
					GroupBox{
						Title:  "Security",
						Layout: Grid{Columns: 2},
						Children: []Widget{
							Label{Text: "API Token:"},
							Composite{
								Layout: HBox{},
								Children: []Widget{
									LineEdit{
										AssignTo: &tokenEdit,
										ReadOnly: true,
										Text:     config.APIToken,
									},
									PushButton{
										Text: "Copy",
										OnClicked: func() {
											if err := walk.Clipboard().SetText(config.APIToken); err == nil {
												walk.MsgBox(dlg, "Success", "Token copied to clipboard!", walk.MsgBoxIconInformation)
											}
										},
									},
									PushButton{
										Text: "Regenerate",
										OnClicked: func() {
											if walk.MsgBox(dlg, "Confirm", "Are you sure you want to regenerate the API token? This will invalidate the current token.", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion) == walk.DlgCmdYes {
												regenerateToken()
												tokenEdit.SetText(config.APIToken)
											}
										},
									},
								},
							},
						},
					},
					GroupBox{
						Title:  "Allowed Paths",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Enter one path per line. Use * for wildcards (e.g., C:\\Users\\*\\Projects)"},
							TextEdit{
								AssignTo: &pathsEdit,
								Text:     strings.Join(config.AllowedPaths, "\r\n"),
								MinSize:  Size{0, 150},
							},
						},
					},
					GroupBox{
						Title:  "Allowed Commands",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Allowed commands: " + strings.Join(config.AllowedCmds, ", ")},
							PushButton{
								Text: "Edit Commands...",
								OnClicked: func() {
									editCommands(dlg)
								},
							},
						},
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "Save",
						OnClicked: func() {
							saveConfig()
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo: &cancelPB,
						Text:     "Cancel",
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}.Run(nil)

	if err != nil {
		log.Printf("Error showing dialog: %v", err)
	}

	updateStatus()
}

func editCommands(owner walk.Form) {
	var dlg *walk.Dialog
	var cmdsEdit *walk.TextEdit

	Dialog{
		AssignTo: &dlg,
		Title:    "Edit Allowed Commands",
		MinSize:  Size{400, 300},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "Enter one command per line:"},
			TextEdit{
				AssignTo: &cmdsEdit,
				Text:     strings.Join(config.AllowedCmds, "\r\n"),
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "OK",
						OnClicked: func() {
							lines := strings.Split(cmdsEdit.Text(), "\r\n")
							config.AllowedCmds = []string{}
							for _, line := range lines {
								line = strings.TrimSpace(line)
								if line != "" {
									config.AllowedCmds = append(config.AllowedCmds, line)
								}
							}
							dlg.Accept()
						},
					},
					PushButton{
						Text:      "Cancel",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Run(owner)
}

func loadConfig() {
	if config != nil {
		return
	}

	exePath, _ := os.Executable()
	baseDir := filepath.Dir(exePath)
	configPath = filepath.Join(baseDir, "config", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		config = &Config{
			Port:         2223,
			AllowedPaths: []string{"C:\\Dev"},
			AllowedCmds:  []string{"go", "msbuild", "dotnet"},
			LogFile:      "logs\\log.txt",
		}
		return
	}

	config = &Config{}
	json.Unmarshal(data, config)
	
	if config.Port == 0 {
		config.Port = 2223
	}
}

func saveConfig() {
	config.Port = int(portEdit.Value())
	
	paths := strings.Split(pathsEdit.Text(), "\r\n")
	config.AllowedPaths = []string{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			config.AllowedPaths = append(config.AllowedPaths, path)
		}
	}

	data, _ := json.MarshalIndent(config, "", "  ")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	os.WriteFile(configPath, data, 0600)

	walk.MsgBox(mainWindow, "Success", "Configuration saved. Please restart the service for changes to take effect.", walk.MsgBoxIconInformation)
}

func regenerateToken() {
	// Generate new token
	token := generateToken()
	config.APIToken = token
	saveConfig()
}

func generateToken() string {
	cmd := exec.Command("powershell", "-Command", "[guid]::NewGuid().ToString('N')")
	output, _ := cmd.Output()
	return strings.TrimSpace(string(output))
}

func controlService(action string) {
	var cmd *exec.Cmd
	
	switch action {
	case "start":
		cmd = exec.Command("sc", "start", "DevProxy")
	case "stop":
		cmd = exec.Command("sc", "stop", "DevProxy")
	case "restart":
		exec.Command("sc", "stop", "DevProxy").Run()
		cmd = exec.Command("sc", "start", "DevProxy")
	}

	if cmd != nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		err := cmd.Run()
		if err != nil {
			walk.MsgBox(nil, "Error", fmt.Sprintf("Failed to %s service: %v", action, err), walk.MsgBoxIconError)
		}
	}
}

func updateStatus() {
	if statusLabel == nil {
		return
	}

	cmd := exec.Command("sc", "query", "DevProxy")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.Output()
	
	if err != nil {
		statusLabel.SetText("Status: Not Installed")
		return
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "RUNNING") {
		statusLabel.SetText("Status: Running")
	} else if strings.Contains(outputStr, "STOPPED") {
		statusLabel.SetText("Status: Stopped")
	} else {
		statusLabel.SetText("Status: Unknown")
	}
}

func getPort() int {
	if config != nil && config.Port > 0 {
		return config.Port
	}
	return 2223
}


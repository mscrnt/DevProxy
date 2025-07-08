package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
)

type Config struct {
	APIToken      string   `json:"api_token"`
	AllowedCmds   []string `json:"allowed_commands"`
	AllowedPaths  []string `json:"allowed_paths"`
	LogFile       string   `json:"log_file"`
	Port          int      `json:"port"`
}

type RunRequest struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	CWD     string   `json:"cwd"`
}

type RunResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	IP        string   `json:"ip"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
	CWD       string   `json:"cwd"`
	Stdout    string   `json:"stdout"`
	Stderr    string   `json:"stderr"`
	ExitCode  int      `json:"exit_code"`
	Status    string   `json:"status"`
	Reason    string   `json:"reason,omitempty"`
}

type devProxyService struct {
	server *http.Server
}

var (
	config     Config
	logFile    *os.File
	bannedKeys = []string{"reg", "shutdown", "format", "schtasks", "sc", "net", "bcdedit", "diskpart"}
)

func main() {
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("Failed to determine if we are running in an interactive session: %v", err)
	}

	if !isIntSess {
		runService()
		return
	}

	runInteractive()
}

func runInteractive() {
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := initLogging(); err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	defer logFile.Close()

	log.Println("Running in interactive mode...")
	startServer()
}

func runService() {
	err := svc.Run("DevProxy", &devProxyService{})
	if err != nil {
		log.Printf("Service failed: %v", err)
	}
}

func (m *devProxyService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	if err := loadConfig(); err != nil {
		log.Printf("Failed to load config: %v", err)
		return
	}

	if err := initLogging(); err != nil {
		log.Printf("Failed to initialize logging: %v", err)
		return
	}
	defer logFile.Close()

	port := config.Port
	if port == 0 {
		port = 2223
	}
	
	m.server = &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", port),
	}

	http.HandleFunc("/run", authMiddleware(handleRun))

	go func() {
		log.Printf("Starting HTTP server on %s", m.server.Addr)
		logEntry(LogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			IP:        "system",
			Status:    "service_start",
			Reason:    fmt.Sprintf("DevProxy service started on %s", m.server.Addr),
		})

		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Println("Received stop command")
				logEntry(LogEntry{
					Timestamp: time.Now().Format(time.RFC3339),
					IP:        "system",
					Status:    "service_stop",
					Reason:    "DevProxy service stopping",
				})
				break loop
			default:
				log.Printf("Unexpected control request #%d", c)
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	m.server.Close()
	return
}

func startServer() {
	http.HandleFunc("/run", authMiddleware(handleRun))
	
	port := config.Port
	if port == 0 {
		port = 2223
	}
	
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	log.Printf("DevProxy starting on %s", addr)
	logEntry(LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		IP:        "system",
		Status:    "server_start",
		Reason:    fmt.Sprintf("DevProxy started on %s", addr),
	})

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func loadConfig() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(exePath)
	configPath := filepath.Join(baseDir, "config", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return createDefaultConfig(configPath)
		}
		return err
	}

	return json.Unmarshal(data, &config)
}

func createDefaultConfig(path string) error {
	token := generateToken()
	
	config = Config{
		APIToken: token,
		AllowedCmds: []string{
			"go", "msbuild", "signtool", "powershell",
			"dotnet", "gcc", "g++", "make", "cmake",
			"npm", "node", "python", "pip",
		},
		AllowedPaths: []string{
			"C:\\Dev",
			"C:\\Users\\*\\Projects",
			"C:\\Users\\*\\source\\repos",
		},
		LogFile: "logs\\log.txt",
		Port:    2223,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}

	fmt.Printf("Created default config at %s\n", path)
	fmt.Printf("Generated API Token: %s\n", token)
	fmt.Printf("Please save this token securely!\n")

	return nil
}

func generateToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func initLogging() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	baseDir := filepath.Dir(exePath)
	
	logPath := config.LogFile
	if !filepath.IsAbs(logPath) {
		logPath = filepath.Join(baseDir, logPath)
	}
	
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return err
	}

	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	return err
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Admin-Token")
		if token != config.APIToken {
			logEntry(LogEntry{
				Timestamp: time.Now().Format(time.RFC3339),
				IP:        r.RemoteAddr,
				Status:    "auth_failed",
				Reason:    "Invalid or missing token",
			})
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		IP:        r.RemoteAddr,
		Command:   req.Command,
		Args:      req.Args,
		CWD:       req.CWD,
	}

	if err := validateRequest(&req); err != nil {
		entry.Status = "rejected"
		entry.Reason = err.Error()
		logEntry(entry)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	stdout, stderr, exitCode := executeCommand(req)

	entry.Stdout = stdout
	entry.Stderr = stderr
	entry.ExitCode = exitCode
	entry.Status = "completed"
	logEntry(entry)

	resp := RunResponse{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func validateRequest(req *RunRequest) error {
	if !isCommandAllowed(req.Command) {
		return fmt.Errorf("command '%s' is not allowed", req.Command)
	}

	if !isPathAllowed(req.CWD) {
		return fmt.Errorf("working directory '%s' is not in allowed paths", req.CWD)
	}

	fullCmd := req.Command + " " + strings.Join(req.Args, " ")
	for _, banned := range bannedKeys {
		if strings.Contains(strings.ToLower(fullCmd), banned) {
			return fmt.Errorf("command contains banned keyword: %s", banned)
		}
	}

	for _, arg := range req.Args {
		if strings.Contains(arg, "..") {
			return fmt.Errorf("path traversal detected in arguments")
		}
		
		if isRestrictedPath(arg) {
			return fmt.Errorf("argument contains restricted path: %s", arg)
		}
	}

	return nil
}

func isCommandAllowed(cmd string) bool {
	cmd = strings.ToLower(filepath.Base(cmd))
	cmd = strings.TrimSuffix(cmd, ".exe")
	
	for _, allowed := range config.AllowedCmds {
		if strings.ToLower(allowed) == cmd {
			return true
		}
	}
	return false
}

func isPathAllowed(path string) bool {
	if path == "" {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, allowed := range config.AllowedPaths {
		if strings.Contains(allowed, "*") {
			pattern := strings.ReplaceAll(allowed, "\\", "/")
			pattern = strings.ReplaceAll(pattern, "*", ".*")
			testPath := strings.ReplaceAll(absPath, "\\", "/")
			
			if matched, _ := filepath.Match(pattern, testPath); matched {
				return true
			}
		} else {
			if strings.HasPrefix(strings.ToLower(absPath), strings.ToLower(allowed)) {
				return true
			}
		}
	}
	
	return false
}

func isRestrictedPath(path string) bool {
	restricted := []string{
		"C:\\Windows",
		"C:\\Program Files",
		"C:\\Program Files (x86)",
		"C:\\ProgramData",
		"C:\\System",
	}

	absPath, _ := filepath.Abs(path)
	lowerPath := strings.ToLower(absPath)

	for _, r := range restricted {
		if strings.HasPrefix(lowerPath, strings.ToLower(r)) {
			return true
		}
	}
	
	return false
}

func executeCommand(req RunRequest) (string, string, int) {
	cmd := exec.Command(req.Command, req.Args...)
	cmd.Dir = req.CWD

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return "", err.Error(), 1
	}

	stdoutBytes, _ := io.ReadAll(stdout)
	stderrBytes, _ := io.ReadAll(stderr)

	err := cmd.Wait()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return string(stdoutBytes), string(stderrBytes), exitCode
}

func logEntry(entry LogEntry) {
	if logFile == nil {
		return
	}
	data, _ := json.Marshal(entry)
	logFile.Write(data)
	logFile.Write([]byte("\n"))
	logFile.Sync()
}
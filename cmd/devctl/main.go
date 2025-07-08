package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIToken string `json:"api_token"`
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

func main() {
	var (
		token   string
		cwd     string
		verbose bool
	)

	flag.StringVar(&token, "token", "", "API token (reads from config if not provided)")
	flag.StringVar(&cwd, "cwd", "", "Working directory (uses current directory if not provided)")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.Parse()

	if flag.NArg() < 1 {
		printUsage()
		os.Exit(1)
	}

	command := flag.Arg(0)
	args := flag.Args()[1:]

	if token == "" {
		var err error
		token, err = loadToken()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not load token: %v\n", err)
			fmt.Fprintf(os.Stderr, "Use -token flag or ensure config.json exists\n")
			os.Exit(1)
		}
	}

	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not get current directory: %v\n", err)
			os.Exit(1)
		}
	}

	if verbose {
		fmt.Printf("Command: %s\n", command)
		fmt.Printf("Args: %v\n", args)
		fmt.Printf("CWD: %s\n", cwd)
		fmt.Printf("Token: %s...\n", token[:8])
		fmt.Println()
	}

	req := RunRequest{
		Command: command,
		Args:    args,
		CWD:     cwd,
	}

	resp, err := executeCommand(token, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if resp.Stdout != "" {
		fmt.Print(resp.Stdout)
	}

	if resp.Stderr != "" {
		fmt.Fprint(os.Stderr, resp.Stderr)
	}

	os.Exit(resp.ExitCode)
}

func printUsage() {
	fmt.Println("devctl - DevProxy CLI client")
	fmt.Println()
	fmt.Println("Usage: devctl [flags] <command> [args...]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -token string   API token (reads from config if not provided)")
	fmt.Println("  -cwd string     Working directory (uses current directory if not provided)")
	fmt.Println("  -v              Verbose output")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  devctl go version")
	fmt.Println("  devctl -cwd C:\\Dev\\MyApp go build -o app.exe")
	fmt.Println("  devctl -token YOUR_TOKEN powershell -Command Get-Date")
}

func loadToken() (string, error) {
	configPaths := []string{
		"config/config.json",
		"../../config/config.json",
		filepath.Join(os.Getenv("USERPROFILE"), ".devproxy", "config.json"),
	}

	for _, path := range configPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var config Config
		if err := json.Unmarshal(data, &config); err != nil {
			continue
		}

		if config.APIToken != "" {
			return config.APIToken, nil
		}
	}

	return "", fmt.Errorf("no config file found with API token")
}

func executeCommand(token string, req RunRequest) (*RunResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", "http://127.0.0.1:2223/run", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Admin-Token", token)

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %d: %s", httpResp.StatusCode, strings.TrimSpace(string(body)))
	}

	var resp RunResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &resp, nil
}
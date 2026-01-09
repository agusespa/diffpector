package evaluation

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type ServerManager struct {
	llamaServerPath string
	port            int
	extraArgs       []string // Additional arguments for llama-server
	cmd             *exec.Cmd
	cancel          context.CancelFunc
}

type ModelServerConfig struct {
	Name      string `json:"name"`
	ModelPath string `json:"model_path"`
	BaseURL   string `json:"base_url"`
	Port      int    `json:"port"`
}

func NewServerManager(llamaServerPath string, port int, extraArgs ...string) *ServerManager {
	return &ServerManager{
		llamaServerPath: llamaServerPath,
		port:            port,
		extraArgs:       extraArgs,
	}
}

func (sm *ServerManager) StartServer(modelPath string) error {
	if sm.cmd != nil {
		return fmt.Errorf("server is already running")
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", modelPath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sm.cancel = cancel

	args := []string{"-m", modelPath, "--port", fmt.Sprintf("%d", sm.port)}
	args = append(args, sm.extraArgs...)

	sm.cmd = exec.CommandContext(ctx, sm.llamaServerPath, args...)

	// Redirect output to log file instead of terminal
	logFile, err := os.Create(fmt.Sprintf("/tmp/llama-server-%d.log", sm.port))
	if err != nil {
		sm.cancel()
		sm.cmd = nil
		sm.cancel = nil
		return fmt.Errorf("failed to create log file: %w", err)
	}
	sm.cmd.Stdout = logFile
	sm.cmd.Stderr = logFile

	if err := sm.cmd.Start(); err != nil {
		_ = logFile.Close()
		sm.cancel()
		sm.cmd = nil
		sm.cancel = nil
		return fmt.Errorf("failed to start llama-server: %w", err)
	}

	fmt.Printf("llama-server output: /tmp/llama-server-%d.log\n", sm.port)

	// Wait for server to be ready (increased timeout for large models)
	if err := sm.waitForServer(120 * time.Second); err != nil {
		sm.StopServer()
		return fmt.Errorf("server failed to become ready: %w", err)
	}

	return nil
}

func (sm *ServerManager) StopServer() {
	if sm.cmd == nil {
		return
	}

	if sm.cancel != nil {
		sm.cancel()
	}

	done := make(chan error, 1)
	go func() {
		done <- sm.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Force kill if it doesn't stop gracefully
		if sm.cmd.Process != nil {
			_ = sm.cmd.Process.Kill()
		}
	case <-done:
		// Process exited
	}

	sm.cmd = nil
	sm.cancel = nil

	// Give it a moment to fully release the port
	time.Sleep(2 * time.Second)
}

func (sm *ServerManager) waitForServer(timeout time.Duration) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	deadline := time.Now().Add(timeout)
	healthURL := fmt.Sprintf("http://localhost:%d/health", sm.port)

	fmt.Printf("Waiting for server to load model (timeout: %v)...\n", timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			// Only consider 200 OK as ready - 503 means still loading
			if resp.StatusCode == http.StatusOK {
				fmt.Println("Server is ready!")
				// Give it a bit more time to stabilize
				time.Sleep(3 * time.Second)
				return nil
			}
			if resp.StatusCode == http.StatusServiceUnavailable {
				fmt.Print(".")
			}
		}

		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("server did not become ready within %v", timeout)
}

func (sm *ServerManager) IsRunning() bool {
	return sm.cmd != nil && sm.cmd.Process != nil
}

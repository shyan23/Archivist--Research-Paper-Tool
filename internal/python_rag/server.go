package python_rag

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Server manages the Python RAG API server
type Server struct {
	pythonPath string
	port       int
	cmd        *exec.Cmd
	client     *PythonRAGClient
}

// NewServer creates a new Python RAG server manager
func NewServer(port int) *Server {
	if port <= 0 {
		port = 8000
	}

	pythonPath := findPythonExecutable()

	return &Server{
		pythonPath: pythonPath,
		port:       port,
		client:     NewPythonRAGClient(fmt.Sprintf("http://localhost:%d", port)),
	}
}

// Start starts the Python RAG API server
func (s *Server) Start() error {
	log.Printf("🚀 Starting Python RAG API server on port %d...", s.port)

	// Find python_rag directory
	pythonRAGDir, err := s.findPythonRAGDir()
	if err != nil {
		return fmt.Errorf("python_rag directory not found: %w", err)
	}

	// Check if requirements are installed
	if err := s.checkRequirements(pythonRAGDir); err != nil {
		log.Printf("⚠️  Python dependencies may not be installed")
		log.Printf("   Run: cd %s && pip install -r requirements.txt", pythonRAGDir)
	}

	// Start the server process
	s.cmd = exec.Command(
		s.pythonPath,
		"-m", "python_rag.cli",
		"server",
		"--port", fmt.Sprintf("%d", s.port),
		"--host", "0.0.0.0",
	)

	// Set working directory
	s.cmd.Dir = filepath.Dir(pythonRAGDir)

	// Set environment
	s.cmd.Env = os.Environ()

	// Redirect output
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr

	// Start the process
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Python server: %w", err)
	}

	log.Printf("  Python server process started (PID: %d)", s.cmd.Process.Pid)

	// Wait for server to be ready
	if err := s.waitForReady(30 * time.Second); err != nil {
		s.Stop()
		return fmt.Errorf("server failed to start: %w", err)
	}

	log.Println("✅ Python RAG API server is ready!")

	return nil
}

// Stop stops the Python RAG API server
func (s *Server) Stop() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	log.Println("🛑 Stopping Python RAG API server...")

	// Try graceful shutdown first
	if err := s.cmd.Process.Signal(os.Interrupt); err != nil {
		// Force kill if graceful shutdown fails
		s.cmd.Process.Kill()
	}

	// Wait for process to exit
	s.cmd.Wait()

	log.Println("✅ Python RAG API server stopped")

	return nil
}

// Client returns the HTTP client for the server
func (s *Server) Client() *PythonRAGClient {
	return s.client
}

// waitForReady waits for the server to be ready
func (s *Server) waitForReady(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	log.Print("  Waiting for server to be ready")

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for server")
		case <-ticker.C:
			if err := s.client.HealthCheck(context.Background()); err == nil {
				return nil
			}
			log.Print(".")
		}
	}
}

// findPythonRAGDir finds the python_rag directory
func (s *Server) findPythonRAGDir() (string, error) {
	// Try current directory
	if _, err := os.Stat("python_rag"); err == nil {
		return "python_rag", nil
	}

	// Try parent directory
	if _, err := os.Stat("../python_rag"); err == nil {
		return "../python_rag", nil
	}

	// Try from executable location
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		pythonRAGPath := filepath.Join(exeDir, "python_rag")
		if _, err := os.Stat(pythonRAGPath); err == nil {
			return pythonRAGPath, nil
		}
	}

	return "", fmt.Errorf("python_rag directory not found")
}

// checkRequirements checks if Python requirements are installed
func (s *Server) checkRequirements(pythonRAGDir string) error {
	cmd := exec.Command(
		s.pythonPath,
		"-c",
		"import chromadb; import sentence_transformers; import fastapi",
	)
	cmd.Dir = pythonRAGDir

	return cmd.Run()
}

// findPythonExecutable finds the Python executable
func findPythonExecutable() string {
	// Try python3 first
	if path, err := exec.LookPath("python3"); err == nil {
		return path
	}

	// Try python
	if path, err := exec.LookPath("python"); err == nil {
		return path
	}

	// Default based on OS
	if runtime.GOOS == "windows" {
		return "python.exe"
	}

	return "python3"
}

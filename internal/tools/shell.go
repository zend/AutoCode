package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ShellInput struct {
	Command string            `json:"command"`
	WorkDir string            `json:"work_dir,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type ShellResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Duration string `json:"duration"`
}

type ShellTool struct {
	baseDir string
}

func NewShellTool(baseDir string) *ShellTool {
	return &ShellTool{baseDir: baseDir}
}

func (t *ShellTool) Name() string {
	return "shell"
}

func (t *ShellTool) Description() string {
	return "Execute shell commands. Supports timeout, working directory, and environment variables."
}

func (t *ShellTool) Execute(ctx context.Context, input string) (string, error) {
	var req ShellInput
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("parse input: %w", err)
	}

	if req.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	workDir := t.baseDir
	if req.WorkDir != "" {
		workDir = filepath.Join(t.baseDir, req.WorkDir)
		workDir = filepath.Clean(workDir)
		if !validatePath(t.baseDir, workDir) {
			return "", fmt.Errorf("work_dir must be within base directory")
		}
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", req.Command)
	cmd.Dir = workDir

	env := os.Environ()
	for k, v := range req.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := ShellResult{
		Command:  req.Command,
		Duration: duration.String(),
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	return t.formatResult(result), nil
}

func (t *ShellTool) formatResult(r ShellResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Command: %s\n", r.Command))
	sb.WriteString(fmt.Sprintf("Exit Code: %d\n", r.ExitCode))
	sb.WriteString(fmt.Sprintf("Duration: %s\n", r.Duration))

	if r.Stdout != "" {
		sb.WriteString("\n--- stdout ---\n")
		sb.WriteString(r.Stdout)
		if !strings.HasSuffix(r.Stdout, "\n") {
			sb.WriteString("\n")
		}
	}

	if r.Stderr != "" {
		sb.WriteString("\n--- stderr ---\n")
		sb.WriteString(r.Stderr)
		if !strings.HasSuffix(r.Stderr, "\n") {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type CursorRunner struct {
	Binary string
}

var (
	flagSupport     map[string]bool
	flagSupportOnce sync.Once
)

func probeFlags(binary string) map[string]bool {
	flagSupportOnce.Do(func() {
		flagSupport = make(map[string]bool)
		out, err := exec.Command(binary, "--help").CombinedOutput()
		if err != nil {
			return
		}
		help := string(out)
		for _, flag := range []string{"--trust", "--force"} {
			if strings.Contains(help, flag) {
				flagSupport[flag] = true
			}
		}
	})
	return flagSupport
}

func (c CursorRunner) Run(ctx context.Context, prompt string, cfg RunConfig) (Result, error) {
	binary := resolveBinary(c.Binary)
	flags := probeFlags(binary)

	args := []string{"-p", "--output-format", "stream-json", "--stream-partial-output"}
	if flags["--trust"] {
		args = append(args, "--trust")
	}
	if flags["--force"] {
		args = append(args, "--force")
	}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	// The agent CLI only accepts "plan" and "ask" for --mode.
	// The default (no --mode flag) is the full agent/coding mode.
	if cfg.Mode == "plan" || cfg.Mode == "ask" {
		args = append(args, "--mode", cfg.Mode)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return Result{}, fmt.Errorf("start cursor runner: %w", err)
	}

	// Stream events to stderr in real-time; accumulate assistant text.
	assistantText, parseErr := streamAndParse(stdoutPipe, os.Stderr)

	if err := cmd.Wait(); err != nil {
		return Result{}, fmt.Errorf("cursor runner failed: %w\n%s", err, strings.TrimSpace(stderr.String()))
	}
	if parseErr != nil {
		return Result{}, fmt.Errorf("parse stream output: %w", parseErr)
	}

	return Result{
		Stdout: assistantText,
		Stderr: stderr.String(),
	}, nil
}

func resolveBinary(binary string) string {
	if binary == "" {
		binary = "agent"
	}
	if _, err := exec.LookPath(binary); err == nil {
		return binary
	}
	if binary == "agent" {
		home, err := os.UserHomeDir()
		if err == nil {
			fallback := filepath.Join(home, ".local", "bin", "agent")
			if _, err := os.Stat(fallback); err == nil {
				return fallback
			}
		}
	}
	return binary
}

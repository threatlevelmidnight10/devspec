package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CursorRunner struct {
	Binary string
}

func (c CursorRunner) Run(ctx context.Context, prompt string, cfg RunConfig) (Result, error) {
	binary := resolveBinary(c.Binary)
	args := []string{"-p", "--trust"}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	if cfg.Mode != "" {
		args = append(args, "--mode", cfg.Mode)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("cursor runner failed: %w\n%s", err, strings.TrimSpace(stderr.String()))
	}

	return Result{
		Stdout: stdout.String(),
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

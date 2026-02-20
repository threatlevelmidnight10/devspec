package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type CursorRunner struct {
	Binary string
}

func (c CursorRunner) Run(ctx context.Context, prompt string, cfg RunConfig) (Result, error) {
	args := make([]string, 0, 4)
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, c.Binary, args...)
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

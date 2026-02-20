package orchestrator

import "context"

type RunConfig struct {
	Model string
}

type Result struct {
	Stdout string
	Stderr string
}

type Runner interface {
	Run(ctx context.Context, prompt string, cfg RunConfig) (Result, error)
}

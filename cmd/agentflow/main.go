package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"agentflow/internal/executor"
	"agentflow/internal/spec"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}

	switch args[0] {
	case "run":
		return runCommand(args[1:])
	default:
		return usageError()
	}
}

func runCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing spec path\n\n%s", usage())
	}

	specPath := args[0]
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var task string
	var dryRun bool
	var noPR bool
	var modelOverride string
	var maxIterOverride int

	fs.StringVar(&task, "task", "", "task description to execute")
	fs.BoolVar(&dryRun, "dry-run", false, "show what would run without changing git state")
	fs.BoolVar(&noPR, "no-pr", false, "skip PR creation regardless of spec output settings")
	fs.StringVar(&modelOverride, "model", "", "override orchestrator model from spec")
	fs.IntVar(&maxIterOverride, "max-iter", 0, "override max iteration constraint")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if task == "" {
		return fmt.Errorf("--task is required")
	}

	s, err := spec.Load(specPath)
	if err != nil {
		return err
	}

	r := executor.Runner{
		Spec: s,
		Opts: executor.Options{
			Task:            task,
			DryRun:          dryRun,
			NoPR:            noPR,
			ModelOverride:   modelOverride,
			MaxIterOverride: maxIterOverride,
		},
	}
	return r.Run(context.Background())
}

func usageError() error {
	return fmt.Errorf("%s", usage())
}

func usage() string {
	return `agentflow - deterministic agent workflow runner

Usage:
  agentflow run <spec.yaml> --task "..." [--dry-run] [--no-pr] [--model override-model] [--max-iter N]
`
}

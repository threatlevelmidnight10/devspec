# agentflow

`agentflow` is a thin Go orchestration CLI for deterministic Cursor-agent workflows.

## Install

Option 1 (recommended): Go install

```bash
go install github.com/threatlevelmidnight10/devspec/cmd/agentflow@latest
```

Option 2: Homebrew (build from this repo)

```bash
brew install --HEAD https://raw.githubusercontent.com/threatlevelmidnight10/devspec/main/Formula/agentflow.rb
```

## MVP Features

- Declarative YAML workflow spec
- Enforced phases: `plan -> implement -> self_review`
- Workspace setup with deterministic branch naming
- Diff-size, test, and iteration guardrails
- Optional auto-commit and PR creation

## Usage

```bash
go run ./cmd/agentflow run examples/schema-migration.yaml \
  --task "Migrate user table to UUID PK"
```

Flags:
- `--dry-run`: parse spec and print phase execution without mutating workspace
- `--no-pr`: skip PR creation
- `--model`: override model from spec
- `--max-iter`: override max iterations

## Expected Tooling

- `git`
- Cursor Agent CLI binary (default: `agent`)
- `gh` (if `output.create_pr: true`)

## Notes

- In MVP, MCP servers are declarative only. `agentflow` does not start/stop MCP servers.
- Spec-relative paths are resolved from the spec file directory.

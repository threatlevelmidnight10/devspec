# devspec

Deterministic AI coding workflows for any repo. Define your workflow as a YAML spec, point it at the [Cursor Agent CLI](https://docs.cursor.com/agent/cli), and let it run — with guardrails.

```
devspec run workflow.yaml --task "Add pagination to /api/users"
```

```
created branch: agent/api-pagination-20260220-120000
==> phase: plan
  🤖 Model: Auto
  📖 Reading: src/api/users.go
     ✅ Read 142 lines
  🎯 Finished in 12.3s
==> phase: implement
  ✏️  Writing: src/api/users.go
     ✅ Created 168 lines (4821 bytes)
  🔧 Running: go test ./...
     ✅ Exit 0
  🎯 Finished in 45.1s
==> phase: self_review
  🎯 Finished in 8.7s
changes committed
pull request created
devspec completed successfully on branch agent/api-pagination-20260220-120000
```

---

## Install

```bash
# Go
go install github.com/threatlevelmidnight10/devspec/cmd/devspec@v0.1.0

# or Homebrew
brew tap threatlevelmidnight10/devspec https://github.com/threatlevelmidnight10/devspec
brew install devspec
```

## Prerequisites

| Tool | Required | Why |
|------|----------|-----|
| `git` | ✅ | Branch management, diffing |
| [Cursor Agent CLI](https://docs.cursor.com/agent/cli) (`agent`) | ✅ | Runs the AI agent |
| `gh` | Only if `output.create_pr: true` | Creates pull requests |

Make sure `agent` is in your PATH:
```bash
agent --version   # should print something like 2026.02.13-xxxxx
```

---

## Quick Start

### 1. Create the spec file

Create a `devspec.yaml` in your repo root (or anywhere — paths are relative to the spec file):

```yaml
version: 0.1
name: my-workflow
description: What this workflow does

orchestrator:
  type: cursor
  model: auto
  binary: agent

workspace:
  base_branch: main
  branch_prefix: agent/
  auto_commit: true

agents:
  planner:
    system_prompt: .devspec/agents/planner.md
    read_only: true

  implementer:
    system_prompt: .devspec/agents/implementer.md
    read_only: false

execution:
  phases:
    - plan
    - implement
    - self_review

constraints:
  max_iterations: 5
  max_diff_lines: 800
  require_tests: true

output:
  create_pr: true
  pr_template: .devspec/templates/pr.md
```

### 2. Write the agent prompts

Create prompt files referenced by the spec:

**.devspec/agents/planner.md**
```markdown
You are a planning agent. Produce a concise execution plan before implementation.

Requirements:
- Identify impacted files.
- Call out risks and rollback strategy.
- Keep output in sections: Steps, Files, Risks.
```

**.devspec/agents/implementer.md**
```markdown
You are an implementation agent. Apply changes directly in the repository.

Requirements:
- Follow repository conventions.
- Add or update tests when logic changes.
- Keep changes bounded to the requested task.
```

### 3. Create the PR template

**.devspec/templates/pr.md**
```markdown
## Summary

## Changes

## Validation
- [ ] Tests added or updated
```

### 4. Run it

```bash
devspec run devspec.yaml --task "Add rate limiting to the auth endpoint"
```

devspec will:
1. Checkout `base_branch` and pull latest
2. Create a new branch (`agent/my-workflow-20260220-120000`)
3. Run each phase sequentially (`plan → implement → self_review`)
4. Enforce constraints (diff size, test presence, iteration limit)
5. Auto-commit and create a PR

---

## Spec Reference

### `orchestrator`
| Field | Default | Description |
|-------|---------|-------------|
| `type` | — | Must be `cursor` |
| `model` | — | Model name or `auto` |
| `binary` | `agent` | Path to Cursor Agent CLI binary |

### `workspace`
| Field | Default | Description |
|-------|---------|-------------|
| `base_branch` | `main` | Branch to fork from |
| `branch_prefix` | `agent/` | Prefix for created branches |
| `auto_commit` | `false` | Commit changes after phases complete |

### `context`
| Field | Default | Description |
|-------|---------|-------------|
| `include_repo_tree` | `false` | Pass repo file tree to the agent |
| `include_git_diff` | `false` | Pass current git diff to the agent |

### `agents`
Define `planner` and/or `implementer` agents:
| Field | Description |
|-------|-------------|
| `system_prompt` | Path to the agent's system prompt (relative to spec file) |
| `read_only` | Whether the agent should only read (used for plan phase) |

### `skills`
List of markdown files injected into agent prompts as additional context:
```yaml
skills:
  - skills/coding_patterns.md
  - skills/repo_rules.md
```

### `execution.phases`
Ordered list of phases to run. Allowed values: `plan`, `implement`, `self_review`.

### `constraints`
| Field | Default | Description |
|-------|---------|-------------|
| `max_iterations` | `5` | Max mutation phases (implement + self_review combined) |
| `max_diff_lines` | `800` | Max diff lines per mutation phase |
| `require_tests` | `false` | Fail if mutation phase doesn't touch test files |

### `output`
| Field | Default | Description |
|-------|---------|-------------|
| `create_pr` | `false` | Create a GitHub PR via `gh` |
| `pr_template` | — | Path to PR body template (required if `create_pr: true`) |

---

## CLI Flags

```
devspec run <spec.yaml> --task "..." [flags]
```

| Flag | Description |
|------|-------------|
| `--task` | **(required)** Task description for the agent |
| `--dry-run` | Parse spec and print phases without running anything |
| `--no-pr` | Skip PR creation even if spec says `create_pr: true` |
| `--model` | Override the model from the spec |
| `--max-iter` | Override `constraints.max_iterations` |

---

## How It Works

```
devspec run spec.yaml --task "..."
        │
        ├─ parse spec.yaml
        ├─ git checkout main && git pull
        ├─ git checkout -b agent/<name>-<timestamp>
        │
        ├─ phase: plan
        │    └─ agent -p --mode plan "<prompt>"
        │         └─ streams: 🤖📖🎯
        │
        ├─ phase: implement
        │    └─ agent -p "<prompt with plan output>"
        │         └─ streams: 🤖📖✏️🔧🎯
        │    └─ validate: diff size, test files
        │
        ├─ phase: self_review
        │    └─ agent -p "<prompt with diff>"
        │         └─ streams: 🤖📖✏️🔧🎯
        │    └─ validate: diff size
        │
        ├─ git add . && git commit
        └─ gh pr create
```

---

## Examples

See [`examples/schema-migration.yaml`](examples/schema-migration.yaml) for a complete working spec.

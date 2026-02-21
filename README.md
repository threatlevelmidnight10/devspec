# devspec

Specify exactly how AI should modify your codebase using a versioned YAML workflow, so every run produces consistent, reviewable changes across the team, point it at the [Cursor Agent CLI](https://docs.cursor.com/agent/cli), and let it run — with guardrails. 

```
devspec run workflow.yaml --task "Add pagination to /api/users"
```

```
created branch: agent/api-pagination-20260220-120000
==> step: plan
  🤖 Model: Auto
  📖 Reading: src/api/users.go
     ✅ Read 142 lines
  🎯 Finished in 12.3s
==> step: implement
  ✏️  Writing: src/api/users.go
     ✅ Created 168 lines (4821 bytes)
  🔧 Running: go test ./...
     ✅ Exit 0
  🎯 Finished in 45.1s
==> step: self_review
  🎯 Finished in 8.7s
changes committed
pull request created
devspec completed successfully on branch agent/api-pagination-20260220-120000
```

---

## Install

```bash
# Go
go install github.com/threatlevelmidnight10/devspec/cmd/devspec@latest

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
model: auto

workspace:
  base_branch: main
  branch_prefix: agent/
  auto_commit: true

agents:
  planner:
    prompt: |
      You are a planning agent. Produce a concise execution plan.
      Output in sections: Steps, Files, Risks.
    read_only: true

  implementer:
    prompt: |
      You are an implementation agent. Apply changes directly in the repository.
      Follow conventions and keep changes bounded to the requested task.
    read_only: false

skills:
  - |
    Prefer small, reviewable changes.
    Add tests when behavior changes.

steps:
  - name: plan
    agent: planner
    mode: plan
  - name: implement
    agent: implementer
  - name: test
    run: make test
    allow_failure: true
  - name: self_review
    agent: implementer

constraints:
  max_iterations: 5
  max_diff_lines: 800
  require_tests: true

output:
  create_pr: true
  pr_template: .devspec/templates/pr.md
```

### 2. (Optional) Use prompt files instead of inline

Prompts can also be file paths. If the value is a single line and the file exists, devspec reads it:

```yaml
agents:
  planner:
    prompt: .devspec/agents/planner.md
  implementer:
    prompt: .devspec/agents/implementer.md
```

### 3. Create the PR template (if using `create_pr: true`)

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
3. Run each step sequentially (`plan → implement → test → self_review`)
4. Enforce constraints (diff size, test presence, iteration limit)
5. Auto-commit and create a PR

---

## Spec Reference

### `model`
Default model for all agents. Individual agents can override with `agents.<name>.model`.

### `binary`
| Field | Default | Description |
|-------|---------|-------------|
| `binary` | `agent` | Path to Cursor Agent CLI binary |

### `workspace`
| Field | Default | Description |
|-------|---------|-------------|
| `base_branch` | `main` | Branch to fork from |
| `branch_prefix` | `agent/` | Prefix for created branches |
| `auto_commit` | `false` | Commit changes after steps complete |
| `repos` | current dir | List of repos to operate on (see below) |

#### Path resolution

All relative paths in the spec (repo paths, prompt files, skill files) are resolved **relative to the spec file's directory**, not your current working directory.

You can run `devspec` from anywhere — only the spec file location matters:
```bash
# All of these work the same way:
devspec run /path/to/spec.yaml --task "..."
cd /path/to && devspec run spec.yaml --task "..."
cd /somewhere/else && devspec run /path/to/spec.yaml --task "..."
```

#### Single repo (default)

If you omit `repos`, devspec treats the spec file's directory as a single git repo:

```yaml
# spec.yaml lives inside your repo
version: 0.1
name: my-workflow
model: auto

# no repos needed — devspec uses the directory containing this file
agents:
  implementer:
    prompt: |
      You are an implementation agent.

steps:
  - name: implement
    agent: implementer
```

#### Multi-repo

For cross-repo workflows, use `repos`. Paths are relative to the spec file:

```
my-services/                <-- spec file lives here
  migration-spec.yaml
  new-service/                  <-- repo 1
  middleware/                   <-- repo 2
  legacy-api/                   <-- repo 3
```

```yaml
# my-services/migration-spec.yaml
workspace:
  repos:
    - name: new-service
      path: new-service              # relative to spec file dir
    - name: middleware
      path: middleware
    - name: legacy-api
      path: legacy-api
      base_branch: master            # this repo uses master, not main
```

For multi-repo setups, devspec generates a temporary `.code-workspace` file so Cursor can see all repos in a single workspace.

### `context`
| Field | Default | Description |
|-------|---------|-------------|
| `include_repo_tree` | `true` | Pass repo file tree to the agent |
| `include_git_diff` | `false` | Pass current git diff to the agent |

### `agents`
Define named agents that steps can reference:
| Field | Description |
|-------|-------------|
| `prompt` | Inline prompt text or path to a prompt file |
| `model` | Optional model override for this agent |
| `read_only` | Whether the agent should only read (used for plan mode) |

### `skills`
List of inline text blocks and/or file paths injected into agent prompts as additional context:
```yaml
skills:
  - |
    Keep changes small and reviewable.
  - skills/coding_patterns.md
  - skills/repo_rules.md
```

### `steps`
Ordered list of workflow steps. Each step must define exactly one of:
- `agent`: run a named agent (`mode` optional: `plan`, `ask`, `agent`, or empty)
- `run`: run a shell command (`allow_failure` optional, defaults to `false`)

```yaml
steps:
  - name: plan
    agent: planner
    mode: plan
  - name: implement
    agent: implementer
  - name: test
    run: make test
    allow_failure: true
  - name: review
    agent: reviewer
```

### `constraints`
| Field | Default | Description |
|-------|---------|-------------|
| `max_iterations` | `5` | Max code-writing agent steps (everything except `mode: plan`) |
| `max_diff_lines` | `800` | Max total `git diff` lines (added + removed) after each agent step |
| `require_tests` | `false` | Fail if an agent step changes code but doesn't touch any test file |

`max_iterations` counts every agent step that can write code. Plan-mode steps are excluded. Example with `max_iterations: 3`:

```
step: plan        → plan mode, NOT counted
step: implement   → counter = 1 ✅
step: self_review → counter = 2 ✅
step: fix         → counter = 3 ✅
step: another_fix → counter = 4 ❌ ABORT
```

`max_diff_lines` and `require_tests` are checked **after** each code-writing agent step finishes, by inspecting `git diff` across all repos. If the constraints are violated, devspec aborts immediately.

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
| `--dry-run` | Parse spec and print steps without running anything |
| `--no-pr` | Skip PR creation even if spec says `create_pr: true` |
| `--model` | Override the model from the spec |
| `--max-iter` | Override `constraints.max_iterations` |
| `--keep-workspace` | Skips cleanup of the temporary multi-repo `.code-workspace` directory |

---

## How It Works

```
devspec run spec.yaml --task "..."
        │
        ├─ parse spec.yaml
        ├─ git checkout main && git pull
        ├─ git checkout -b agent/<name>-<timestamp>
        │
        ├─ step: plan
        │    └─ agent -p --mode plan "<prompt>"
        │
        ├─ step: implement
        │    └─ agent -p "<prompt with plan output>"
        │    └─ validate: diff size, test files
        │
        ├─ step: test
        │    └─ sh -c "make test"
        │
        ├─ step: self_review
        │    └─ agent -p "<prompt with diff>"
        │    └─ validate: diff size
        │
        ├─ git add . && git commit
        └─ gh pr create
```

---

## Examples

See [`examples/schema-migration.yaml`](examples/schema-migration.yaml) and [`examples/multi-repo.yaml`](examples/multi-repo.yaml) for complete working specs.

---

## Roadmap

- **Git Registry**: Share agents, skills, and rules across teams via a git repo (`from: team/agents/planner`).
- **Rules Support**: Inject `.cursorrules` into workspaces from the spec.
- **Multi-Runtime**: Support Claude Code, Windsurf, and other agentic CLIs via `--runtime` flag.
- **`devspec init`**: Scaffold specs from templates.

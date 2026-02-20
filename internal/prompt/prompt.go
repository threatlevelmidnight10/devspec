package prompt

import (
	"fmt"
	"os"
	"strings"

	"github.com/threatlevelmidnight10/devspec/internal/spec"
)

type Inputs struct {
	Spec          *spec.Spec
	Task          string
	PlannerPrompt string
	ImplPrompt    string
	Skills        []string
	RepoTree      string
	GitDiff       string
	PlanOutput    string
	DiffOutput    string
}

func LoadFiles(paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", p, err)
		}
		out = append(out, string(b))
	}
	return out, nil
}

func BuildPlan(in Inputs) string {
	return strings.TrimSpace(fmt.Sprintf(`%s

%s

%s

TASK:
%s

STRICT MODE:
- Do not modify files.
- Output a structured plan only with sections: Steps, Files, Risks.

CONSTRAINTS:
- max_iterations: %d
- max_diff_lines: %d
- require_tests: %t
`,
		header("PLANNER SYSTEM PROMPT", in.PlannerPrompt),
		header("SKILLS", joinBlocks(in.Skills)),
		sharedContext(in),
		in.Task,
		in.Spec.Constraints.MaxIterations,
		in.Spec.Constraints.MaxDiffLines,
		in.Spec.Constraints.RequireTests,
	))
}

func BuildImplement(in Inputs) string {
	return strings.TrimSpace(fmt.Sprintf(`%s

%s

%s

PLAN OUTPUT:
%s

TASK:
%s

CONSTRAINTS:
- Max diff lines: %d
- Tests required: %t
- Produce changes directly in workspace.
`,
		header("IMPLEMENTER SYSTEM PROMPT", in.ImplPrompt),
		header("SKILLS", joinBlocks(in.Skills)),
		sharedContext(in),
		in.PlanOutput,
		in.Task,
		in.Spec.Constraints.MaxDiffLines,
		in.Spec.Constraints.RequireTests,
	))
}

func BuildSelfReview(in Inputs) string {
	return strings.TrimSpace(fmt.Sprintf(`%s

%s

Review the current changes critically and fix issues found.

DIFF:
%s

CHECKLIST:
- Tests exist when required.
- Migration safety.
- Backward compatibility.
- Constraint compliance.
`,
		header("IMPLEMENTER SYSTEM PROMPT", in.ImplPrompt),
		header("SKILLS", joinBlocks(in.Skills)),
		in.DiffOutput,
	))
}

func sharedContext(in Inputs) string {
	parts := []string{}
	if strings.TrimSpace(in.RepoTree) != "" {
		parts = append(parts, header("REPO TREE", in.RepoTree))
	}
	if strings.TrimSpace(in.GitDiff) != "" {
		parts = append(parts, header("CURRENT GIT DIFF", in.GitDiff))
	}
	if len(parts) == 0 {
		return "CONTEXT:\n(none)"
	}
	return strings.Join(parts, "\n\n")
}

func header(title, body string) string {
	if strings.TrimSpace(body) == "" {
		body = "(none)"
	}
	return fmt.Sprintf("%s:\n%s", title, body)
}

func joinBlocks(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, "\n\n---\n\n")
}

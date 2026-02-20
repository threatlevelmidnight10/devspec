package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/threatlevelmidnight10/devspec/internal/gitutil"
	"github.com/threatlevelmidnight10/devspec/internal/orchestrator"
	"github.com/threatlevelmidnight10/devspec/internal/prompt"
	"github.com/threatlevelmidnight10/devspec/internal/spec"
)

var testFilePattern = regexp.MustCompile(`(?i)(^|/)(test|tests)(/|$)|(_test\.|\.test\.|\.spec\.)`)

type Options struct {
	Task            string
	DryRun          bool
	NoPR            bool
	ModelOverride   string
	MaxIterOverride int
}

type Runner struct {
	Spec         *spec.Spec
	Opts         Options
	Orchestrator orchestrator.Runner
	Workdir      string
	Now          func() time.Time
}

type runState struct {
	repoRoot           string
	branchName         string
	model              string
	planOutput         string
	repoTree           string
	gitDiff            string
	plannerPrompt      string
	implementerPrompt  string
	skillBodies        []string
	mutationIterations int
}

func (r *Runner) Run(ctx context.Context) error {
	if r.Spec == nil {
		return errors.New("spec is required")
	}
	if strings.TrimSpace(r.Opts.Task) == "" {
		return errors.New("task is required")
	}

	if r.Now == nil {
		r.Now = time.Now
	}
	if r.Workdir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve workdir: %w", err)
		}
		r.Workdir = wd
	}

	st := &runState{model: r.Spec.EffectiveModel(r.Opts.ModelOverride)}
	if r.Opts.MaxIterOverride > 0 {
		r.Spec.Constraints.MaxIterations = r.Opts.MaxIterOverride
	}
	if r.Orchestrator == nil {
		r.Orchestrator = orchestrator.CursorRunner{Binary: r.Spec.Orchestrator.Binary}
	}

	if err := gitutil.EnsureRepo(ctx, r.Workdir); err != nil {
		return err
	}
	root, err := gitutil.RepoRoot(ctx, r.Workdir)
	if err != nil {
		return err
	}
	st.repoRoot = root

	if err := gitutil.EnsureClean(ctx, st.repoRoot); err != nil {
		return err
	}

	if err := r.loadContent(st); err != nil {
		return err
	}

	if r.Spec.Context.IncludeRepoTree {
		tree, err := gitutil.RepoTree(ctx, st.repoRoot)
		if err != nil {
			return err
		}
		st.repoTree = tree
	}
	if r.Spec.Context.IncludeGitDiff {
		d, err := gitutil.Diff(ctx, st.repoRoot)
		if err != nil {
			return err
		}
		st.gitDiff = d
	}

	if err := r.setupWorkspace(ctx, st); err != nil {
		return err
	}

	for _, phase := range r.Spec.Execution.Phases {
		fmt.Printf("==> phase: %s\n", phase)
		if err := r.runPhase(ctx, st, phase); err != nil {
			return err
		}
	}

	if err := r.finalize(ctx, st); err != nil {
		return err
	}

	fmt.Printf("devspec completed successfully on branch %s\n", st.branchName)
	return nil
}

func (r *Runner) loadContent(st *runState) error {
	planner, hasPlanner := r.Spec.Agents["planner"]
	if hasPlanner {
		b, err := os.ReadFile(r.Spec.ResolvePath(planner.SystemPrompt))
		if err != nil {
			return fmt.Errorf("read planner system prompt: %w", err)
		}
		st.plannerPrompt = string(b)
	}

	impl, hasImpl := r.Spec.Agents["implementer"]
	if hasImpl {
		b, err := os.ReadFile(r.Spec.ResolvePath(impl.SystemPrompt))
		if err != nil {
			return fmt.Errorf("read implementer system prompt: %w", err)
		}
		st.implementerPrompt = string(b)
	}

	if len(r.Spec.Skills) > 0 {
		paths := make([]string, 0, len(r.Spec.Skills))
		for _, p := range r.Spec.Skills {
			paths = append(paths, r.Spec.ResolvePath(p))
		}
		skills, err := prompt.LoadFiles(paths)
		if err != nil {
			return err
		}
		st.skillBodies = skills
	}

	return nil
}

func (r *Runner) setupWorkspace(ctx context.Context, st *runState) error {
	if r.Opts.DryRun {
		current, err := gitutil.CurrentBranch(ctx, st.repoRoot)
		if err != nil {
			return err
		}
		st.branchName = current
		fmt.Println("dry-run enabled: workspace mutations skipped")
		return nil
	}

	if err := gitutil.Checkout(ctx, st.repoRoot, r.Spec.Workspace.BaseBranch); err != nil {
		return err
	}
	if err := gitutil.PullFFOnly(ctx, st.repoRoot); err != nil {
		return err
	}

	st.branchName = makeBranchName(r.Spec.Workspace.BranchPref, r.Spec.Name, r.Now())
	if err := gitutil.CreateBranch(ctx, st.repoRoot, st.branchName); err != nil {
		return err
	}
	fmt.Printf("created branch: %s\n", st.branchName)
	return nil
}

func (r *Runner) runPhase(ctx context.Context, st *runState, phase string) error {
	if r.Opts.DryRun {
		fmt.Printf("dry-run: would execute %s\n", phase)
		return nil
	}

	switch phase {
	case "plan":
		before, err := gitutil.ChangedFiles(ctx, st.repoRoot)
		if err != nil {
			return err
		}
		out, err := r.Orchestrator.Run(ctx, prompt.BuildPlan(prompt.Inputs{
			Spec:          r.Spec,
			Task:          r.Opts.Task,
			PlannerPrompt: st.plannerPrompt,
			Skills:        st.skillBodies,
			RepoTree:      st.repoTree,
			GitDiff:       st.gitDiff,
		}), orchestrator.RunConfig{Model: st.model, Mode: "plan"})
		if err != nil {
			return err
		}
		st.planOutput = out.Stdout
		after, err := gitutil.ChangedFiles(ctx, st.repoRoot)
		if err != nil {
			return err
		}
		if !sameFiles(before, after) {
			return errors.New("plan phase modified files, which is not allowed")
		}
		fmt.Println("plan phase complete")
		return nil

	case "implement":
		if err := r.bumpIteration(st); err != nil {
			return err
		}
		_, err := r.Orchestrator.Run(ctx, prompt.BuildImplement(prompt.Inputs{
			Spec:       r.Spec,
			Task:       r.Opts.Task,
			ImplPrompt: st.implementerPrompt,
			Skills:     st.skillBodies,
			RepoTree:   st.repoTree,
			GitDiff:    st.gitDiff,
			PlanOutput: st.planOutput,
		}), orchestrator.RunConfig{Model: st.model, Mode: "agent"})
		if err != nil {
			return err
		}
		if err := r.validateMutation(ctx, st, true); err != nil {
			return err
		}
		fmt.Println("implement phase complete")
		return nil

	case "self_review":
		if err := r.bumpIteration(st); err != nil {
			return err
		}
		d, err := gitutil.Diff(ctx, st.repoRoot)
		if err != nil {
			return err
		}
		_, err = r.Orchestrator.Run(ctx, prompt.BuildSelfReview(prompt.Inputs{
			Spec:       r.Spec,
			ImplPrompt: st.implementerPrompt,
			Skills:     st.skillBodies,
			DiffOutput: d,
		}), orchestrator.RunConfig{Model: st.model, Mode: "agent"})
		if err != nil {
			return err
		}
		if err := r.validateMutation(ctx, st, false); err != nil {
			return err
		}
		fmt.Println("self_review phase complete")
		return nil

	default:
		return fmt.Errorf("unsupported phase %q", phase)
	}
}

func (r *Runner) bumpIteration(st *runState) error {
	st.mutationIterations++
	if st.mutationIterations > r.Spec.Constraints.MaxIterations {
		return fmt.Errorf("max iterations exceeded (%d)", r.Spec.Constraints.MaxIterations)
	}
	return nil
}

func (r *Runner) validateMutation(ctx context.Context, st *runState, requireDiff bool) error {
	d, err := gitutil.Diff(ctx, st.repoRoot)
	if err != nil {
		return err
	}
	files, err := gitutil.ChangedFiles(ctx, st.repoRoot)
	if err != nil {
		return err
	}
	if requireDiff && len(files) == 0 {
		return errors.New("phase produced no diff")
	}

	lines := gitutil.DiffLineCount(d)
	if lines > r.Spec.Constraints.MaxDiffLines {
		return fmt.Errorf("diff line limit exceeded (%d > %d)", lines, r.Spec.Constraints.MaxDiffLines)
	}

	if r.Spec.Constraints.RequireTests && len(files) > 0 && !hasTestFile(files) {
		return errors.New("constraints.require_tests is true but no test files were modified")
	}

	return nil
}

func (r *Runner) finalize(ctx context.Context, st *runState) error {
	if r.Opts.DryRun {
		return nil
	}

	files, err := gitutil.ChangedFiles(ctx, st.repoRoot)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no changes generated")
	}

	if r.Spec.Workspace.AutoCommit {
		if err := gitutil.AddAll(ctx, st.repoRoot); err != nil {
			return err
		}
		msg := fmt.Sprintf("devspec: %s", r.Spec.Name)
		if err := gitutil.Commit(ctx, st.repoRoot, msg); err != nil {
			return err
		}
		fmt.Println("changes committed")
	}

	if r.Spec.Output.CreatePR && !r.Opts.NoPR {
		if err := gitutil.Push(ctx, st.repoRoot, st.branchName); err != nil {
			return err
		}
		if err := createPR(ctx, st.repoRoot, r.Spec, st.branchName, r.Opts.Task); err != nil {
			return err
		}
	}

	return nil
}

func createPR(ctx context.Context, repoRoot string, s *spec.Spec, branch, task string) error {
	bodyPath := s.ResolvePath(s.Output.PRTemplate)
	if _, err := os.Stat(bodyPath); err != nil {
		return fmt.Errorf("pr template not found: %w", err)
	}

	title := fmt.Sprintf("devspec: %s", task)
	cmd := exec.CommandContext(ctx, "gh", "pr", "create",
		"--base", s.Workspace.BaseBranch,
		"--head", branch,
		"--title", title,
		"--body-file", bodyPath,
	)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh pr create failed: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	prURL := strings.TrimSpace(string(out))
	fmt.Printf("pull request created: %s\n", prURL)
	return nil
}

func makeBranchName(prefix, name string, ts time.Time) string {
	cleanPrefix := strings.TrimSuffix(strings.TrimSpace(prefix), "/")
	if cleanPrefix == "" {
		cleanPrefix = "agent"
	}
	safe := sanitizeName(name)
	return fmt.Sprintf("%s/%s-%s", cleanPrefix, safe, ts.UTC().Format("20060102-150405"))
}

func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	re := regexp.MustCompile(`[^a-z0-9._-]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-./")
	if s == "" {
		return "run"
	}
	return s
}

func hasTestFile(files []string) bool {
	for _, f := range files {
		if testFilePattern.MatchString(filepath.ToSlash(f)) {
			return true
		}
	}
	return false
}

func sameFiles(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	am := make(map[string]struct{}, len(a))
	for _, f := range a {
		am[f] = struct{}{}
	}
	for _, f := range b {
		if _, ok := am[f]; !ok {
			return false
		}
	}
	return true
}

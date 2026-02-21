package spec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var allowedStepModes = map[string]struct{}{
	"":      {},
	"agent": {},
	"plan":  {},
	"ask":   {},
}

type Spec struct {
	Version     string           `yaml:"version" json:"version"`
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description" json:"description"`
	Model       string           `yaml:"model" json:"model"`
	Workspace   Workspace        `yaml:"workspace" json:"workspace"`
	Context     Context          `yaml:"context" json:"context"`
	Agents      map[string]Agent `yaml:"agents" json:"agents"`
	Skills      []string         `yaml:"skills" json:"skills"`
	Steps       []Step           `yaml:"steps" json:"steps"`
	Constraints Constraints      `yaml:"constraints" json:"constraints"`
	Output      Output           `yaml:"output" json:"output"`
	Binary      string           `yaml:"binary" json:"binary"`
	SourcePath  string           `yaml:"-" json:"-"`
	SourceDir   string           `yaml:"-" json:"-"`
}

type Workspace struct {
	BaseBranch string     `yaml:"base_branch" json:"base_branch"`
	BranchPref string     `yaml:"branch_prefix" json:"branch_prefix"`
	AutoCommit bool       `yaml:"auto_commit" json:"auto_commit"`
	Repos      []RepoSpec `yaml:"repos" json:"repos"`
}

type RepoSpec struct {
	Name       string `yaml:"name" json:"name"`
	Path       string `yaml:"path" json:"path"`
	BaseBranch string `yaml:"base_branch" json:"base_branch"`
}

type Context struct {
	IncludeRepoTree bool `yaml:"include_repo_tree" json:"include_repo_tree"`
	IncludeGitDiff  bool `yaml:"include_git_diff" json:"include_git_diff"`
	IncludeUntrack  bool `yaml:"include_untracked" json:"include_untracked"`
}

type Agent struct {
	Prompt   string `yaml:"prompt" json:"prompt"`
	Model    string `yaml:"model" json:"model"`
	ReadOnly bool   `yaml:"read_only" json:"read_only"`
}

type Step struct {
	Name         string `yaml:"name" json:"name"`
	Agent        string `yaml:"agent" json:"agent"`
	Mode         string `yaml:"mode" json:"mode"`
	Run          string `yaml:"run" json:"run"`
	AllowFailure bool   `yaml:"allow_failure" json:"allow_failure"`
	Retry        int    `yaml:"retry" json:"retry"`
}

type Constraints struct {
	MaxIterations int  `yaml:"max_iterations" json:"max_iterations"`
	MaxDiffLines  int  `yaml:"max_diff_lines" json:"max_diff_lines"`
	RequireTests  bool `yaml:"require_tests" json:"require_tests"`
}

type Output struct {
	CreatePR   bool   `yaml:"create_pr" json:"create_pr"`
	PRTemplate string `yaml:"pr_template" json:"pr_template"`
}

func Load(path string) (*Spec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}

	var s Spec
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("decode spec: %w", err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve spec path: %w", err)
	}
	s.SourcePath = abs
	s.SourceDir = filepath.Dir(abs)

	applyDefaults(&s)
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func applyDefaults(s *Spec) {
	if s.Binary == "" {
		s.Binary = "agent"
	}
	if s.Workspace.BaseBranch == "" {
		s.Workspace.BaseBranch = "main"
	}
	if s.Workspace.BranchPref == "" {
		s.Workspace.BranchPref = "agent/"
	}
	if len(s.Workspace.Repos) == 0 {
		s.Workspace.Repos = []RepoSpec{
			{Name: "default", Path: ".", BaseBranch: s.Workspace.BaseBranch},
		}
	} else {
		for i := range s.Workspace.Repos {
			if s.Workspace.Repos[i].BaseBranch == "" {
				s.Workspace.Repos[i].BaseBranch = s.Workspace.BaseBranch
			}
		}
	}
	s.Context.IncludeRepoTree = true
	if s.Constraints.MaxIterations == 0 {
		s.Constraints.MaxIterations = 5
	}
	if s.Constraints.MaxDiffLines == 0 {
		s.Constraints.MaxDiffLines = 800
	}
}

func (s *Spec) Validate() error {
	if s.Version == "" {
		return errors.New("spec.version is required")
	}
	if s.Name == "" {
		return errors.New("spec.name is required")
	}
	if strings.TrimSpace(s.Model) == "" {
		return errors.New("model is required")
	}
	if len(s.Steps) == 0 {
		return errors.New("steps is required")
	}
	for i, step := range s.Steps {
		if strings.TrimSpace(step.Name) == "" {
			return fmt.Errorf("steps[%d].name is required", i)
		}
		if step.Retry < 0 {
			return fmt.Errorf("steps[%d].retry cannot be negative", i)
		}
		hasAgent := strings.TrimSpace(step.Agent) != ""
		hasRun := strings.TrimSpace(step.Run) != ""
		if hasAgent == hasRun {
			return fmt.Errorf("steps[%d] must define exactly one of agent or run", i)
		}
		if _, ok := allowedStepModes[strings.TrimSpace(step.Mode)]; !ok {
			return fmt.Errorf("steps[%d].mode %q is invalid; allowed: plan, ask, agent, or empty", i, step.Mode)
		}
		if hasRun && strings.TrimSpace(step.Mode) != "" {
			return fmt.Errorf("steps[%d].mode is only valid for agent steps", i)
		}
		if hasAgent {
			ag, ok := s.Agents[step.Agent]
			if !ok {
				return fmt.Errorf("steps[%d].agent %q is not defined in agents", i, step.Agent)
			}
			if strings.TrimSpace(ag.Prompt) == "" {
				return fmt.Errorf("agents.%s.prompt is required", step.Agent)
			}
		}
	}
	if s.Constraints.MaxIterations <= 0 {
		return errors.New("constraints.max_iterations must be > 0")
	}
	if s.Constraints.MaxDiffLines <= 0 {
		return errors.New("constraints.max_diff_lines must be > 0")
	}
	if s.Output.CreatePR && strings.TrimSpace(s.Output.PRTemplate) == "" {
		return errors.New("output.pr_template is required when output.create_pr is true")
	}
	return nil
}

func (s *Spec) EffectiveModel(override string) string {
	if override != "" {
		return override
	}
	return s.Model
}

func (s *Spec) EffectiveAgentModel(agentName, override string) string {
	if override != "" {
		return override
	}
	ag, ok := s.Agents[agentName]
	if ok && strings.TrimSpace(ag.Model) != "" {
		return strings.TrimSpace(ag.Model)
	}
	return s.Model
}

func (s *Spec) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.SourceDir, path)
}

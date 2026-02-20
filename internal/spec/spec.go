package spec

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var allowedPhases = map[string]struct{}{
	"plan":        {},
	"implement":   {},
	"self_review": {},
}

type Spec struct {
	Version      string           `yaml:"version" json:"version"`
	Name         string           `yaml:"name" json:"name"`
	Description  string           `yaml:"description" json:"description"`
	Orchestrator Orchestrator     `yaml:"orchestrator" json:"orchestrator"`
	Workspace    Workspace        `yaml:"workspace" json:"workspace"`
	Context      Context          `yaml:"context" json:"context"`
	Agents       map[string]Agent `yaml:"agents" json:"agents"`
	Skills       []string         `yaml:"skills" json:"skills"`
	MCP          MCPConfig        `yaml:"mcp" json:"mcp"`
	Execution    Execution        `yaml:"execution" json:"execution"`
	Constraints  Constraints      `yaml:"constraints" json:"constraints"`
	Output       Output           `yaml:"output" json:"output"`
	SourcePath   string           `yaml:"-" json:"-"`
	SourceDir    string           `yaml:"-" json:"-"`
}

type Orchestrator struct {
	Type   string `yaml:"type" json:"type"`
	Model  string `yaml:"model" json:"model"`
	Binary string `yaml:"binary" json:"binary"`
}

type Workspace struct {
	BaseBranch string `yaml:"base_branch" json:"base_branch"`
	BranchPref string `yaml:"branch_prefix" json:"branch_prefix"`
	AutoCommit bool   `yaml:"auto_commit" json:"auto_commit"`
}

type Context struct {
	IncludeRepoTree bool `yaml:"include_repo_tree" json:"include_repo_tree"`
	IncludeGitDiff  bool `yaml:"include_git_diff" json:"include_git_diff"`
	IncludeUntrack  bool `yaml:"include_untracked" json:"include_untracked"`
}

type Agent struct {
	SystemPrompt string `yaml:"system_prompt" json:"system_prompt"`
	ReadOnly     bool   `yaml:"read_only" json:"read_only"`
}

type MCPConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Servers []string `yaml:"servers" json:"servers"`
}

type Execution struct {
	Phases []string `yaml:"phases" json:"phases"`
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

	root, err := parseYAML(b)
	if err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	j, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("convert YAML: %w", err)
	}

	var s Spec
	if err := json.Unmarshal(j, &s); err != nil {
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
	if s.Orchestrator.Binary == "" {
		s.Orchestrator.Binary = "agent"
	}
	if s.Workspace.BaseBranch == "" {
		s.Workspace.BaseBranch = "main"
	}
	if s.Workspace.BranchPref == "" {
		s.Workspace.BranchPref = "agent/"
	}
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
	if s.Orchestrator.Type == "" {
		return errors.New("orchestrator.type is required")
	}
	if s.Orchestrator.Type != "cursor" {
		return fmt.Errorf("unsupported orchestrator.type %q", s.Orchestrator.Type)
	}
	if s.Orchestrator.Model == "" {
		return errors.New("orchestrator.model is required")
	}
	if len(s.Execution.Phases) == 0 {
		return errors.New("execution.phases is required")
	}
	for _, p := range s.Execution.Phases {
		if _, ok := allowedPhases[p]; !ok {
			return fmt.Errorf("invalid phase %q; allowed: plan, implement, self_review", p)
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
	if needsPhase(s.Execution.Phases, "plan") {
		if _, ok := s.Agents["planner"]; !ok {
			return errors.New("agents.planner is required when plan phase is enabled")
		}
	}
	if needsAnyPhase(s.Execution.Phases, "implement", "self_review") {
		if _, ok := s.Agents["implementer"]; !ok {
			return errors.New("agents.implementer is required when implement or self_review phase is enabled")
		}
	}

	for name, ag := range s.Agents {
		if strings.TrimSpace(ag.SystemPrompt) == "" {
			return fmt.Errorf("agents.%s.system_prompt is required", name)
		}
	}

	return nil
}

func (s *Spec) EffectiveModel(override string) string {
	if override != "" {
		return override
	}
	return s.Orchestrator.Model
}

func (s *Spec) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.SourceDir, path)
}

func needsPhase(phases []string, wanted string) bool {
	for _, p := range phases {
		if p == wanted {
			return true
		}
	}
	return false
}

func needsAnyPhase(phases []string, wanted ...string) bool {
	set := make(map[string]struct{}, len(wanted))
	for _, w := range wanted {
		set[w] = struct{}{}
	}
	for _, p := range phases {
		if _, ok := set[p]; ok {
			return true
		}
	}
	return false
}

package spec

import (
	"testing"
)

func TestValidateRejectsInvalidPhase(t *testing.T) {
	s := Spec{
		Version: "0.1",
		Name:    "x",
		Orchestrator: Orchestrator{
			Type:  "cursor",
			Model: "claude-3.5-sonnet",
		},
		Execution: Execution{Phases: []string{"shipit"}},
		Agents: map[string]Agent{
			"planner": {SystemPrompt: "planner.md"},
		},
		Constraints: Constraints{MaxIterations: 5, MaxDiffLines: 100},
	}

	if err := s.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEffectiveModelUsesOverride(t *testing.T) {
	s := Spec{Orchestrator: Orchestrator{Model: "base"}}
	if got := s.EffectiveModel("override"); got != "override" {
		t.Fatalf("expected override, got %q", got)
	}
}

func TestSpecParsing(t *testing.T) {
	s, err := Load("../../examples/schema-migration.yaml")
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	got := s.Agents["implementer"].SystemPrompt
	if got != "agents/implementer.md" {
		t.Fatalf("unexpected implementer prompt: %q", got)
	}
}

package spec

import (
	"testing"
)

func TestValidateRejectsInvalidStepMode(t *testing.T) {
	s := Spec{
		Version: "0.1",
		Name:    "x",
		Model:   "gpt-5.3-codex",
		Steps: []Step{
			{Name: "plan", Agent: "planner", Mode: "shipit"},
		},
		Agents: map[string]Agent{
			"planner": {Prompt: "planner prompt"},
		},
		Constraints: Constraints{MaxIterations: 5, MaxDiffLines: 100},
	}

	if err := s.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestEffectiveModelUsesOverride(t *testing.T) {
	s := Spec{Model: "base"}
	if got := s.EffectiveModel("override"); got != "override" {
		t.Fatalf("expected override, got %q", got)
	}
}

func TestEffectiveAgentModelUsesAgentValue(t *testing.T) {
	s := Spec{
		Model: "default-model",
		Agents: map[string]Agent{
			"planner": {Model: "planner-model"},
		},
	}
	if got := s.EffectiveAgentModel("planner", ""); got != "planner-model" {
		t.Fatalf("expected planner-model, got %q", got)
	}
}

func TestEffectiveAgentModelFallsBackToSpecModel(t *testing.T) {
	s := Spec{
		Model: "default-model",
		Agents: map[string]Agent{
			"planner": {Prompt: "do stuff"},
		},
	}
	if got := s.EffectiveAgentModel("planner", ""); got != "default-model" {
		t.Fatalf("expected default-model, got %q", got)
	}
}

func TestSpecParsing(t *testing.T) {
	s, err := Load("../../examples/schema-migration.yaml")
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	got := s.Agents["implementer"].Prompt
	if got != "agents/implementer.md" {
		t.Fatalf("unexpected implementer prompt: %q", got)
	}
	if len(s.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(s.Steps))
	}
}

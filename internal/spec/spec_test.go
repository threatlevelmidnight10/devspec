package spec

import (
	"os"
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

func TestLoadParsesNestedAgents(t *testing.T) {
	b, err := os.ReadFile("../../examples/schema-migration.yaml")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	root, err := parseYAML(b)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	agents, ok := root["agents"].(map[string]any)
	if !ok {
		t.Fatalf("agents missing or wrong type: %#v", root["agents"])
	}
	impl, ok := agents["implementer"].(map[string]any)
	if !ok {
		t.Fatalf("implementer missing or wrong type: %#v", agents["implementer"])
	}
	if impl["system_prompt"] != "agents/implementer.md" {
		t.Fatalf("unexpected parser value: %#v", impl["system_prompt"])
	}

	s, err := Load("../../examples/schema-migration.yaml")
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	got := s.Agents["implementer"].SystemPrompt
	if got != "agents/implementer.md" {
		t.Fatalf("unexpected implementer prompt: %q", got)
	}
}

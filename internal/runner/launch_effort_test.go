package runner

import (
	"reflect"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

func TestBuildCopilotArgsWithEffort(t *testing.T) {
	got := buildCopilotArgs("high", []string{"--yolo", "continue"})
	want := []string{"--reasoning-effort", "high", "--yolo", "continue"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestBuildCodexArgsWithEffort(t *testing.T) {
	_, got := BuildCodexArgs(&config.Profile{}, "gpt-5", "high")
	if got[len(got)-2] != "--config" || got[len(got)-1] != `model_reasoning_effort="high"` {
		t.Fatalf("got %#v", got)
	}
}

func TestBuildClaudeEnvWithEffortAndSubModel(t *testing.T) {
	env := BuildClaudeEnv(&config.Profile{}, "model", "high", "claude-haiku-4-5")
	for _, want := range []string{"CLAUDE_CODE_ALWAYS_ENABLE_EFFORT=1", "CLAUDE_CODE_EFFORT_LEVEL=high", "CLAUDE_CODE_SUBAGENT_MODEL=claude-haiku-4-5"} {
		found := false
		for _, entry := range env {
			if entry == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing %q in %#v", want, env)
		}
	}
}

func TestBuildPiArgsWithEffort(t *testing.T) {
	got := buildPiArgs("gpt-5", "high", []string{"--approve"})
	want := []string{"--model", "gpt-5", "--thinking", "high", "--approve"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

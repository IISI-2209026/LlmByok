package cmd

import (
	"strings"
	"testing"
)

func TestValidateLaunchEffort_TargetAllowlists(t *testing.T) {
	for _, tc := range []struct {
		target string
		effort string
		valid  bool
	}{
		{"copilot", "max", true}, {"codex", "xhigh", true},
		{"codex-app", "none", true}, {"claude", "low", true},
		{"pi", "off", true}, {"claude", "none", false},
	} {
		t.Run(tc.target+"/"+tc.effort, func(t *testing.T) {
			err := validateLaunchEffort(tc.target, tc.effort)
			if tc.valid && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.valid && (err == nil || !strings.Contains(err.Error(), tc.target) || !strings.Contains(err.Error(), tc.effort)) {
				t.Fatalf("error = %v, want target and effort", err)
			}
		})
	}
}

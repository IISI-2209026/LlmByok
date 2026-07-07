package cmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/config"
)

// TestResolveModel_ModelFlagOverridesCandidates 驗證 --model 非空時一律使用該值，
// 不論候選數量，且不顯示互動選單。
func TestResolveModel_ModelFlagOverridesCandidates(t *testing.T) {
	profile := &config.Profile{Name: "p", Models: []string{"a", "b"}}
	var stdin bytes.Buffer
	var stdout, stderr bytes.Buffer
	got, err := resolveModelForLaunch(profile, "x", &stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("resolveModelForLaunch: %v", err)
	}
	if got != "x" {
		t.Errorf("got %q, want %q (--model overrides)", got, "x")
	}
	if stdout.Len() != 0 {
		t.Errorf("should not render menu when --model given, got: %s", stdout.String())
	}
}

// TestResolveModel_SingleCandidateUsedDirectly 驗證單一候選模型時直接使用，
// 不讀取 stdin、不顯示選單。
func TestResolveModel_SingleCandidateUsedDirectly(t *testing.T) {
	profile := &config.Profile{Name: "p", Models: []string{"gpt-4o"}}
	var stdin bytes.Buffer
	var stdout, stderr bytes.Buffer
	got, err := resolveModelForLaunch(profile, "", &stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("resolveModelForLaunch: %v", err)
	}
	if got != "gpt-4o" {
		t.Errorf("got %q, want %q (single candidate)", got, "gpt-4o")
	}
	if stdout.Len() != 0 {
		t.Errorf("should not render menu for single candidate, got: %s", stdout.String())
	}
}

// TestResolveModel_EmptyCandidatesRejected 驗證空候選清單回傳 errExit
// 並提示 byok config set-models。
func TestResolveModel_EmptyCandidatesRejected(t *testing.T) {
	profile := &config.Profile{Name: "p", Models: nil}
	var stdin bytes.Buffer
	var stdout, stderr bytes.Buffer
	_, err := resolveModelForLaunch(profile, "", &stdin, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "byok config set-models") {
		t.Errorf("stderr should hint byok config set-models, got: %s", stderr.String())
	}
}

// TestResolveModel_MultipleCandidatesNonTtyRejected 驗證多候選且 stdin
// 非終端機時回傳 errExit 並提示 --model。
func TestResolveModel_MultipleCandidatesNonTtyRejected(t *testing.T) {
	orig := isTerminal
	t.Cleanup(func() { isTerminal = orig })
	isTerminal = func(int) bool { return false }
	profile := &config.Profile{Name: "p", Models: []string{"a", "b"}}
	stdin := bytes.NewBufferString("")
	var stdout, stderr bytes.Buffer
	_, err := resolveModelForLaunch(profile, "", stdin, &stdout, &stderr)
	if err != errExit {
		t.Fatalf("err = %v, want errExit", err)
	}
	if !strings.Contains(stderr.String(), "--model") {
		t.Errorf("stderr should hint --model, got: %s", stderr.String())
	}
}

// TestResolveModel_MultipleCandidatesInteractiveSelects 驗證多候選且 stdin
// 為終端機時呼叫互動選單，並以使用者選取項回傳。
func TestResolveModel_MultipleCandidatesInteractiveSelects(t *testing.T) {
	orig := stdinTerminalCheck
	t.Cleanup(func() { stdinTerminalCheck = orig })
	stdinTerminalCheck = func(io.Reader) bool { return true }
	profile := &config.Profile{Name: "p", Models: []string{"gpt-4o", "gpt-4o-mini"}}
	// 模擬按下 Enter（選第一個）。
	stdin := bytes.NewBufferString("\r")
	var stdout, stderr bytes.Buffer
	got, err := resolveModelForLaunch(profile, "", stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("resolveModelForLaunch: %v", err)
	}
	if got != "gpt-4o" {
		t.Errorf("got %q, want %q (first candidate on Enter)", got, "gpt-4o")
	}
}

// TestResolveModel_MultipleCandidatesInteractiveDownThenEnter 驗證向下鍵後
// Enter 選擇第二個候選。
func TestResolveModel_MultipleCandidatesInteractiveDownThenEnter(t *testing.T) {
	orig := stdinTerminalCheck
	t.Cleanup(func() { stdinTerminalCheck = orig })
	stdinTerminalCheck = func(io.Reader) bool { return true }
	profile := &config.Profile{Name: "p", Models: []string{"gpt-4o", "gpt-4o-mini"}}
	stdin := bytes.NewBufferString("\x1b[B\r")
	var stdout, stderr bytes.Buffer
	got, err := resolveModelForLaunch(profile, "", stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("resolveModelForLaunch: %v", err)
	}
	if got != "gpt-4o-mini" {
		t.Errorf("got %q, want %q (second candidate after Down+Enter)", got, "gpt-4o-mini")
	}
}
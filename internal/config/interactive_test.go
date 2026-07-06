package config

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrompter_PromptString(t *testing.T) {
	in := strings.NewReader("openai-official\n")
	var out bytes.Buffer
	p := &Prompter{In: in, Out: &out, IsTTY: func(int) bool { return false }}
	got, err := p.PromptString("profile 名稱")
	if err != nil {
		t.Fatalf("PromptString: %v", err)
	}
	if got != "openai-official" {
		t.Errorf("got %q, want %q", got, "openai-official")
	}
	if !strings.Contains(out.String(), "profile 名稱") {
		t.Errorf("output missing label, got: %s", out.String())
	}
}

func TestPrompter_PromptDefault_UsesDefaultOnEmpty(t *testing.T) {
	in := strings.NewReader("\n")
	var out bytes.Buffer
	p := &Prompter{In: in, Out: &out, IsTTY: func(int) bool { return false }}
	got, err := p.PromptDefault("provider", "openai")
	if err != nil {
		t.Fatalf("PromptDefault: %v", err)
	}
	if got != "openai" {
		t.Errorf("got %q, want default %q", got, "openai")
	}
}

func TestPrompter_PromptDefault_UsesInputWhenProvided(t *testing.T) {
	in := strings.NewReader("anthropic\n")
	var out bytes.Buffer
	p := &Prompter{In: in, Out: &out, IsTTY: func(int) bool { return false }}
	got, err := p.PromptDefault("provider", "openai")
	if err != nil {
		t.Fatalf("PromptDefault: %v", err)
	}
	if got != "anthropic" {
		t.Errorf("got %q, want %q", got, "anthropic")
	}
}

func TestPrompter_PromptSecret_NonTTYReadsLine(t *testing.T) {
	in := strings.NewReader("sk-xxxx\n")
	var out bytes.Buffer
	p := &Prompter{In: in, Out: &out, IsTTY: func(int) bool { return false }}
	got, err := p.PromptSecret("API key")
	if err != nil {
		t.Fatalf("PromptSecret: %v", err)
	}
	if got != "sk-xxxx" {
		t.Errorf("got %q, want %q", got, "sk-xxxx")
	}
}

func TestPrompter_PromptSecret_EmptyReturnsEmpty(t *testing.T) {
	in := strings.NewReader("\n")
	var out bytes.Buffer
	p := &Prompter{In: in, Out: &out, IsTTY: func(int) bool { return false }}
	got, err := p.PromptSecret("API key")
	if err != nil {
		t.Fatalf("PromptSecret: %v", err)
	}
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestPrompter_PromptChoice_DefaultOnEmpty(t *testing.T) {
	in := strings.NewReader("\n")
	var out bytes.Buffer
	p := &Prompter{In: in, Out: &out, IsTTY: func(int) bool { return false }}
	got, err := p.PromptChoice("金鑰儲存", []string{"keychain", "plaintext"}, "keychain")
	if err != nil {
		t.Fatalf("PromptChoice: %v", err)
	}
	if got != "keychain" {
		t.Errorf("got %q, want default keychain", got)
	}
}

func TestPrompter_PromptChoice_CaseInsensitive(t *testing.T) {
	in := strings.NewReader("Plaintext\n")
	var out bytes.Buffer
	p := &Prompter{In: in, Out: &out, IsTTY: func(int) bool { return false }}
	got, err := p.PromptChoice("金鑰儲存", []string{"keychain", "plaintext"}, "keychain")
	if err != nil {
		t.Fatalf("PromptChoice: %v", err)
	}
	if got != "plaintext" {
		t.Errorf("got %q, want plaintext", got)
	}
}

func TestPrompter_PromptChoice_InvalidRejects(t *testing.T) {
	in := strings.NewReader("foo\n")
	var out bytes.Buffer
	p := &Prompter{In: in, Out: &out, IsTTY: func(int) bool { return false }}
	if _, err := p.PromptChoice("金鑰儲存", []string{"keychain", "plaintext"}, "keychain"); err == nil {
		t.Fatal("expected error for invalid choice, got nil")
	}
}

// TestPrompter_MultiplePromptsShareBuffer 確認連續提示共用單一 bufio.Reader，
// 不會因每次新建 reader 而丟失緩衝中的後續行。
func TestPrompter_MultiplePromptsShareBuffer(t *testing.T) {
	in := strings.NewReader("alpha\nbeta\ngamma\n")
	var out bytes.Buffer
	p := &Prompter{In: in, Out: &out, IsTTY: func(int) bool { return false }}
	a, err := p.PromptString("第一")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	b, err := p.PromptString("第二")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	c, err := p.PromptString("第三")
	if err != nil {
		t.Fatalf("third: %v", err)
	}
	if a != "alpha" || b != "beta" || c != "gamma" {
		t.Errorf("got %q,%q,%q want alpha,beta,gamma", a, b, c)
	}
}

func TestApplyProfileUpdates_KeepsUnspecified(t *testing.T) {
	p := &Profile{Name: "p", Provider: "openai", APIBase: "https://x", DefaultModel: "m"}
	ApplyProfileUpdates(p, nil, nil, nil)
	if p.Provider != "openai" || p.APIBase != "https://x" || p.DefaultModel != "m" {
		t.Errorf("fields changed unexpectedly: %+v", p)
	}
}

func TestApplyProfileUpdates_OverwritesSpecified(t *testing.T) {
	p := &Profile{Name: "p", Provider: "openai", APIBase: "https://x", DefaultModel: "m"}
	newProvider := "anthropic"
	newBase := "https://y"
	ApplyProfileUpdates(p, &newProvider, &newBase, nil)
	if p.Provider != "anthropic" {
		t.Errorf("Provider = %q, want anthropic", p.Provider)
	}
	if p.APIBase != "https://y" {
		t.Errorf("APIBase = %q, want https://y", p.APIBase)
	}
	if p.DefaultModel != "m" {
		t.Errorf("DefaultModel = %q, want unchanged m", p.DefaultModel)
	}
}
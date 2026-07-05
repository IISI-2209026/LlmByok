package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/version"
)

// TestVersionCmdOutput 驗證 version 子指令輸出格式為 "byok version <Version>"。
func TestVersionCmdOutput(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "0.1.0"

	cmd := NewVersionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.Run(cmd, []string{})

	got := buf.String()
	want := "byok version 0.1.0\n"
	if got != want {
		t.Errorf("輸出應為 %q，得到 %q", want, got)
	}
}

// TestVersionCmdDefaultDev 驗證未注入版本時輸出 "byok version dev"。
func TestVersionCmdDefaultDev(t *testing.T) {
	orig := version.Version
	defer func() { version.Version = orig }()
	version.Version = "dev"

	cmd := NewVersionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.Run(cmd, []string{})

	got := strings.TrimSpace(buf.String())
	want := "byok version dev"
	if got != want {
		t.Errorf("輸出應為 %q，得到 %q", want, got)
	}
}
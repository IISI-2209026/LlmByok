package config

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestSelectModel_NonTerminalReturnsError 驗證 stdin 非終端機時
// SelectModel 回傳錯誤且不讀取輸入。
func TestSelectModel_NonTerminalReturnsError(t *testing.T) {
	var in bytes.Buffer
	var out bytes.Buffer
	_, err := SelectModel([]string{"a", "b"}, &in, &out, func(int) bool { return false })
	if err == nil {
		t.Fatal("expected error for non-terminal stdin, got nil")
	}
}

// TestSelectModel_SingleCandidateNoMenu 驗證單一候選模型時直接回傳該模型，
// 不顯示選單、不讀取輸入。
func TestSelectModel_SingleCandidateNoMenu(t *testing.T) {
	var in bytes.Buffer
	var out bytes.Buffer
	got, err := SelectModel([]string{"gpt-4o"}, &in, &out, func(int) bool { return true })
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if got != "gpt-4o" {
		t.Errorf("got %q, want %q", got, "gpt-4o")
	}
	if out.Len() != 0 {
		t.Errorf("expected no menu output for single candidate, got: %s", out.String())
	}
}

// TestSelectModel_EmptyCandidatesError 驗證空候選清單回傳錯誤。
func TestSelectModel_EmptyCandidatesError(t *testing.T) {
	var in bytes.Buffer
	var out bytes.Buffer
	_, err := SelectModel(nil, &in, &out, func(int) bool { return true })
	if err == nil {
		t.Fatal("expected error for empty candidates, got nil")
	}
}

// TestSelectModel_EnterSelectsFirst 驗證終端機下按 Enter 直接選擇第一個
// （預設反白）候選模型。
func TestSelectModel_EnterSelectsFirst(t *testing.T) {
	// 模擬使用者按下 Enter（\r）。
	in := bytes.NewBufferString("\r")
	var out bytes.Buffer
	got, err := SelectModel([]string{"gpt-4o", "gpt-4o-mini"}, in, &out, func(int) bool { return true })
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if got != "gpt-4o" {
		t.Errorf("got %q, want %q (first candidate on Enter)", got, "gpt-4o")
	}
}

// TestSelectModel_DownThenEnterSelectsSecond 驗證按下向下鍵後 Enter 選擇第二個候選。
// 向下鍵以常見的 ANSI 序列 \x1b[B 表示。
func TestSelectModel_DownThenEnterSelectsSecond(t *testing.T) {
	in := bytes.NewBufferString("\x1b[B\r")
	var out bytes.Buffer
	got, err := SelectModel([]string{"gpt-4o", "gpt-4o-mini"}, in, &out, func(int) bool { return true })
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if got != "gpt-4o-mini" {
		t.Errorf("got %q, want %q (second candidate after Down+Enter)", got, "gpt-4o-mini")
	}
}

// TestSelectModel_UpWrapsToLast 驗證在第一個候選上按向上鍵會繞回最後一個候選。
func TestSelectModel_UpWrapsToLast(t *testing.T) {
	in := bytes.NewBufferString("\x1b[A\r")
	var out bytes.Buffer
	got, err := SelectModel([]string{"a", "b", "c"}, in, &out, func(int) bool { return true })
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	if got != "c" {
		t.Errorf("got %q, want %q (wrap to last on Up)", got, "c")
	}
}

// TestSelectModel_CtrlCancelsSelection 驗證按下 Ctrl-C（\x03）時回傳取消錯誤。
func TestSelectModel_CtrlCancelsSelection(t *testing.T) {
	in := bytes.NewBufferString("\x03")
	var out bytes.Buffer
	_, err := SelectModel([]string{"a", "b"}, in, &out, func(int) bool { return true })
	if !errors.Is(err, errSelectionCancelled) {
		t.Fatalf("err = %v, want errSelectionCancelled", err)
	}
}

// TestSelectModel_EscapeCancelsSelection 驗證單獨按下 ESC（非方向鍵序列）
// 時回傳取消錯誤。
func TestSelectModel_EscapeCancelsSelection(t *testing.T) {
	// ESC 後接一個非 '[' 字元，使其不構成方向鍵序列 → 視為取消。
	in := bytes.NewBufferString("\x1bz")
	var out bytes.Buffer
	_, err := SelectModel([]string{"a", "b"}, in, &out, func(int) bool { return true })
	if !errors.Is(err, errSelectionCancelled) {
		t.Fatalf("err = %v, want errSelectionCancelled", err)
	}
}

// TestSelectModel_DisplaysAllCandidates 雖然實際顯示格式由實作決定，
// 但所有候選模型名稱都應出現在輸出中。
func TestSelectModel_DisplaysAllCandidates(t *testing.T) {
	in := bytes.NewBufferString("\r")
	var out bytes.Buffer
	_, err := SelectModel([]string{"gpt-4o", "gpt-4o-mini", "o3"}, in, &out, func(int) bool { return true })
	if err != nil {
		t.Fatalf("SelectModel failed: %v", err)
	}
	for _, m := range []string{"gpt-4o", "gpt-4o-mini", "o3"} {
		if !strings.Contains(out.String(), m) {
			t.Errorf("output missing candidate %q; got: %s", m, out.String())
		}
	}
}

// 確保 errSelectNonTerminal 與 ErrNoCandidates 為可被 errors.Is 比對的已命名錯誤。
func TestSelectModel_ErrorsAreNamed(t *testing.T) {
	var in bytes.Buffer
	var out bytes.Buffer
	_, err := SelectModel(nil, &in, &out, func(int) bool { return true })
	if !errors.Is(err, ErrNoCandidates) {
		t.Errorf("empty candidates error should match ErrNoCandidates, got %v", err)
	}
	_, err = SelectModel([]string{"a", "b"}, &in, &out, func(int) bool { return false })
	if !errors.Is(err, ErrNotInteractive) {
		t.Errorf("non-terminal error should match ErrNotInteractive, got %v", err)
	}
}
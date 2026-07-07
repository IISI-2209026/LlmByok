// Package config 的 models.go 提供候選模型的互動式選擇函式，
// 供 `byok launch <target>` 在未帶 --model 且候選模型多個時使用。
//
// SelectModel 以可注入的 io.Reader/io.Writer 與終端機判定函式驅動，
// 使單元測試得以 bytes.Buffer 與模擬按鍵序列驗證，無需真實 TTY。
// 在真實終端機上，呼叫端通常會傳入 os.Stdin/os.Stdout 與 term.IsTerminal。
//
// 真實終端機下的行為：
//   - stdin 切換為 raw mode（term.MakeRaw）以即時讀取按鍵、關閉本地回顯與行緩衝，
//     使方向鍵以 ANSI 序列送達而不被 console 行編輯攔截；離開時還原。
//   - stdout 啟用虛擬終端機（VT）處理（platformEnableVT），使 ANSI 反白與游標
//     控制序列在 Windows console 正確渲染（Unix 終端機原生支援）。
//   - 以「❯ 游標 + 反白」標記選取列，原地重繪。
package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ErrNotInteractive 表示 stdin 非終端機，無法進行互動式模型選擇。
// 呼叫端應改要求使用者以 --model 旗標明確指定模型。
var ErrNotInteractive = errors.New("互動式模型選擇需要終端機；請改用 --model 旗標指定模型")

// ErrNoCandidates 表示候選模型清單為空，無法選擇。
var ErrNoCandidates = errors.New("候選模型清單為空；請先以 `byok config set-models <profile name>` 設定模型")

// 選單導覽使用的 ANSI 控制序列與按鍵。
const (
	ansiDown   = "\x1b[B"
	ansiUp     = "\x1b[A"
	keyEnter   = '\r'
	keyNewline = '\n'
	esc        = '\x1b'
	keyCancel  = 0x03 // Ctrl-C：中斷選擇。
	keyEscape  = 0x1b // ESC（單獨按下，非方向鍵序列）：取消選擇。
)

// makeRaw 亦可被測試替換。預設以 term.MakeRaw 將 fd 切換為 raw mode。
// 回傳的 restore 函式用於還原終端機狀態。
var makeRaw = func(fd int) (restore func() error, err error) {
	old, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}
	return func() error { return term.Restore(fd, old) }, nil
}

// enableVT 嘗試為給定 fd 啟用虛擬終端機處理，使 ANSI 序列在 Windows console
// 正確渲染。回傳的 restore 還原原狀；非 Windows 平台為 no-op。可被測試替換。
var enableVT = platformEnableVT

// SelectModel 在終端機互動環境下讓使用者以上下鍵從 models 中選擇一個模型，
// 按 Enter 確認後回傳所選模型字串。
//
// 行為規則：
//   - models 為空 → 回傳 ErrNoCandidates。
//   - isTerminal 回傳 false → 回傳 ErrNotInteractive（不讀取輸入）。
//   - models 恰一個 → 直接回傳該模型，不顯示選單、不讀取輸入。
//   - models 多個 → 顯示含所有候選的選單，反白第一個；讀取按鍵：
//     向下鍵移動至下一個（在最後一個上則繞回第一個）、向上鍵移動至上一个
//     （在第一個上則繞回最後一個）、Enter 確認目前反白項。
//
// 當 in 為 *os.File（真實終端機）時，進入 raw mode 並啟用 stdout VT 處理；
// 非 *os.File（測試 buffer）時跳過終端機切換，仍以注入的按鍵序列驅動選單，
// 使單元測試無需真實 TTY 即可驗證導覽與選取邏輯。
//
// in/out 可為任意 io.Reader/io.Writer，搭配可注入的 isTerminal 以利測試。
func SelectModel(models []string, in io.Reader, out io.Writer, isTerminal func(fd int) bool) (string, error) {
	if len(models) == 0 {
		return "", ErrNoCandidates
	}
	if len(models) == 1 {
		return models[0], nil
	}
	if !isTerminal(0) {
		return "", ErrNotInteractive
	}

	// 真實終端機（*os.File）才切換 raw/VT；測試 buffer 跳過以維持可注入性。
	stdinFile, stdinIsFile := in.(*os.File)
	stdoutFile, stdoutIsFile := out.(*os.File)
	if stdinIsFile {
		restore, err := makeRaw(int(stdinFile.Fd()))
		if err == nil {
			defer restore()
		}
		// raw mode 失敗時仍以 cooked 模式嘗試讀取（最佳努力）。
	}
	if stdoutIsFile {
		if restore, err := enableVT(int(stdoutFile.Fd())); err == nil {
			defer restore()
		}
	}

	// 渲染初始選單，反白第 0 個候選。
	selected := 0
	renderMenu(out, models, selected)

	r, ok := in.(io.RuneReader)
	if !ok {
		// 以 bufio 包一層 rune reader，避免一次性緩衝讀取造成按鍵遺失。
		r = bufio.NewReader(in)
	}

	for {
		ch, _, err := r.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// 輸入結束未確認：清除選單後回傳目前反白項以避免卡住。
				clearMenu(out, models)
				return models[selected], nil
			}
			return "", fmt.Errorf("讀取選擇按鍵: %w", err)
		}
		switch ch {
		case keyEnter, keyNewline:
			// 確認目前反白項；先清除選單再回傳。
			clearMenu(out, models)
			return models[selected], nil
		case keyCancel:
			// Ctrl-C：清除選單後回傳取消錯誤，呼叫端應以非零結束碼退出。
			clearMenu(out, models)
			return "", errSelectionCancelled
		case esc:
			// 可能為方向鍵序列起始（ESC [ B / ESC [ A）。
			seq, err := readArrow(r)
			if err != nil {
				// 單獨 ESC（非方向鍵）：視為取消。
				clearMenu(out, models)
				return "", errSelectionCancelled
			}
			switch seq {
			case ansiDown:
				selected = (selected + 1) % len(models)
				renderMenu(out, models, selected)
			case ansiUp:
				selected = (selected - 1 + len(models)) % len(models)
				renderMenu(out, models, selected)
			}
		default:
			// 其餘按鍵忽略。
		}
	}
}

// ErrSelectionCancelled 表示使用者以 Ctrl-C 或 ESC 取消互動選擇。
// 呼叫端可判斷此錯誤以提供簡潔的取消提示。
var ErrSelectionCancelled = errors.New("模型選擇已取消")

// errSelectionCancelled 為向後相容的別名（內部使用）。
var errSelectionCancelled = ErrSelectionCancelled

// readArrow 在讀到 ESC 後嘗試讀取方向鍵的剩餘兩個字元（[ 與 A/B/C/D）。
// 回傳完整 ANSI 序列（如 "\x1b[B"）；若不構成方向鍵則回傳空字串與錯誤。
func readArrow(r io.RuneReader) (string, error) {
	bracket, _, err := r.ReadRune()
	if err != nil {
		return "", err
	}
	if bracket != '[' {
		return "", fmt.Errorf("非方向鍵序列")
	}
	arrow, _, err := r.ReadRune()
	if err != nil {
		return "", err
	}
	switch arrow {
	case 'A':
		return ansiUp, nil
	case 'B':
		return ansiDown, nil
	default:
		return "", fmt.Errorf("非上下方向鍵")
	}
}

// renderMenu 清除上一幅選單後重新繪製，以「❯ 游標 + 反白」標記 selected 列。
// 反白以 ANSI 序列 \x1b[7m（reverse video）...\x1b[0m（reset）實作；
// 未選取列以空白取代游標，維持對齊。
func renderMenu(out io.Writer, models []string, selected int) {
	// 先將游標上移並清除已有列，避免重複堆疊。
	clearMenu(out, models)
	var b strings.Builder
	b.WriteString("使用上下鍵選擇模型，按 Enter 確認（Esc 取消）：\n")
	for i, m := range models {
		if i == selected {
			// 反白 + 游標標記選取列。
			b.WriteString("❯ \x1b[7m" + m + "\x1b[0m\n")
		} else {
			b.WriteString("  " + m + "\n")
		}
	}
	fmt.Fprint(out, b.String())
}

// clearMenu 將游標上移 len(models)+1 列並清除，使下一次 renderMenu 覆蓋原圖。
// 行數固定為「標題 1 列 + 候選 N 列」。
func clearMenu(out io.Writer, models []string) {
	lines := len(models) + 1
	// \x1b[<n>A 上移 n 列；\x1b[J 清除游標至螢幕底部。
	fmt.Fprintf(out, "\x1b[%dA\x1b[J", lines)
}
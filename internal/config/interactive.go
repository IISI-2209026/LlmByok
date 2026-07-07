// Package config 的 interactive.go 提供可注入的終端提示器，
// 供 `byok config add`/`update` 互動模式使用。提示器接收 io.Reader/io.Writer
// 與可替換的 terminal 判定函式，使單元測試得以 bytes.Buffer 驅動，無需真實 TTY。
package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// IsTerminalFunc 回報給定檔案描述子是否為終端機。可注入以利測試。
type IsTerminalFunc func(fd int) bool

// DefaultIsTerminal 是預設的 terminal 判定函式，委派給 golang.org/x/term。
var DefaultIsTerminal IsTerminalFunc = term.IsTerminal

// Prompter 驅動互動式欄位提示。In/Out 可為任意 io.Reader/io.Writer，
// IsTTY 決定是否將 stdin 視為終端機（用於非回顯金鑰輸入與非 TTY 失敗判定）。
type Prompter struct {
	In    io.Reader
	Out   io.Writer
	IsTTY IsTerminalFunc

	r *bufio.Reader
}

// IsTerminal 回報提示器的輸入是否被視為終端機。
// 當 In 為 *os.File 時，以 IsTTY 判定其檔案描述子；否則回傳 false
// （buffer/管線輸入不被視為終端機，除非測試注入永真 IsTTY 並搭配 buffer）。
func (p *Prompter) IsTerminal() bool {
	if p.IsTTY == nil {
		return false
	}
	if f, ok := p.In.(*os.File); ok {
		return p.IsTTY(int(f.Fd()))
	}
	return false
}

// PromptString 印出 "label: " 並回傳去除首尾空白的一行輸入。
func (p *Prompter) PromptString(label string) (string, error) {
	fmt.Fprintf(p.Out, "%s: ", label)
	return p.readLine()
}

// PromptDefault 印出 "label [def]: "，回傳輸入值；空白時回傳 def。
func (p *Prompter) PromptDefault(label, def string) (string, error) {
	fmt.Fprintf(p.Out, "%s [%s]: ", label, def)
	line, err := p.readLine()
	if err != nil {
		return "", err
	}
	if line == "" {
		return def, nil
	}
	return line, nil
}

// PromptSecret 讀取金鑰。當輸入為終端機時以 term.ReadPassword 不回顯讀取，
// 否則以行讀取（測試/管線路徑）。空白金鑰回傳空字串。
func (p *Prompter) PromptSecret(label string) (string, error) {
	fmt.Fprintf(p.Out, "%s: ", label)
	if p.IsTerminal() {
		f := p.In.(*os.File)
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(p.Out)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return p.readLine()
}

// PromptChoice 印出 label 與選項清單及預設值，回傳所選選項（不分大小寫比對）；
// 空白輸入回傳 def。未列出的輸入值回傳錯誤。
func (p *Prompter) PromptChoice(label string, options []string, def string) (string, error) {
	fmt.Fprintf(p.Out, "%s (%s) [%s]: ", label, strings.Join(options, "/"), def)
	line, err := p.readLine()
	if err != nil {
		return "", err
	}
	if line == "" {
		return def, nil
	}
	for _, o := range options {
		if strings.EqualFold(line, o) {
			return o, nil
		}
	}
	return "", fmt.Errorf("無效的選項 %q（可選: %s）", line, strings.Join(options, "/"))
}

// reader 懶建立並重用單一 bufio.Reader，避免每次提示新建 reader 造成緩衝資料遺失。
func (p *Prompter) reader() *bufio.Reader {
	if p.r == nil {
		p.r = bufio.NewReader(p.In)
	}
	return p.r
}

// readLine 自共用 reader 讀取一行並去除首尾空白。io.EOF 視為空行結束。
func (p *Prompter) readLine() (string, error) {
	line, err := p.reader().ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
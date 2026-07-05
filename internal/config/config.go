// Package config 負責載入與儲存 byok 設定檔。
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/IISI-2209026/LlmByok/internal/secret"
	"gopkg.in/yaml.v3"
)

// Profile 描述單一 LLM provider 設定。
type Profile struct {
	Name         string `yaml:"name"`
	Provider     string `yaml:"provider"`
	APIBase      string `yaml:"api_base"`
	APIKey       string `yaml:"api_key,omitempty"`
	DefaultModel string `yaml:"default_model"`
}

// Config 是頂層設定結構，持有設定檔清單以及未指定設定檔時
// 使用的設定檔名稱。
type Config struct {
	Profiles       []Profile `yaml:"profiles"`
	DefaultProfile string    `yaml:"default_profile"`
}

// DefaultConfigPath 回傳設定檔的標準位置，位於
// <UserHomeDir>/.byok/config.yaml。
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".byok", "config.yaml"), nil
}

// Load 讀取並解析 path 指向的 YAML 設定檔。檔案不存在時回傳
// 包含路徑的錯誤；解析錯誤時同時提及路徑與底層原因。存在但為空
// 的檔案會回傳非 nil 的 *Config（零值）且不附錯誤。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("設定檔不存在: %w", err)
		}
		return nil, fmt.Errorf("讀取設定檔 %s: %w", path, err)
	}

	cfg := &Config{}
	if len(data) == 0 {
		return cfg, nil
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析設定檔 %s: %w", path, err)
	}
	return cfg, nil
}

// Save 將 cfg 序列化為 YAML 並寫入 path，會建立缺失的上層目錄。
// 檔案以 0600 權限寫入。
func Save(path string, cfg *Config) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("建立設定檔目錄 %q: %w", dir, err)
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化設定檔: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("寫入設定檔 %q: %w", path, err)
	}
	return nil
}

// Source 標識 API 金鑰的來源。
type Source int

const (
	// SourceKeychain 表示金鑰取自 OS keychain。
	SourceKeychain Source = iota
	// SourcePlaintext 表示金鑰取自設定檔明文。
	SourcePlaintext
	// SourceMissing 表示找不到金鑰。
	SourceMissing
)

// String 回傳 Source 的可讀名稱。
func (s Source) String() string {
	switch s {
	case SourceKeychain:
		return "keychain"
	case SourcePlaintext:
		return "plaintext"
	default:
		return "missing"
	}
}

// KeyResolver 解析 Profile 的 API 金鑰來源。
type KeyResolver interface {
	// Resolve 回傳 API 金鑰、其來源以及可能的錯誤。
	Resolve(p Profile) (apiKey string, source Source, err error)
}

// DefaultResolver 先查 keychain，再退回明文。
type DefaultResolver struct{}

// Resolve 依序嘗試 keychain → plaintext → missing。
func (DefaultResolver) Resolve(p Profile) (string, Source, error) {
	if key, err := secret.Load(p.Name); err == nil && key != "" {
		return key, SourceKeychain, nil
	}
	if p.APIKey != "" {
		return p.APIKey, SourcePlaintext, nil
	}
	return "", SourceMissing, fmt.Errorf("profile %q 沒有 API 金鑰（keychain 與明文皆為空）", p.Name)
}

// Resolver 是全域預設解析器，可在測試中被替換。
var Resolver KeyResolver = DefaultResolver{}

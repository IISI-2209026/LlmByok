// Package config 負責載入與儲存 byok 設定檔。
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile 描述單一 LLM provider 設定。
type Profile struct {
	Name         string `yaml:"name"`
	Provider     string `yaml:"provider"`
	APIBase      string `yaml:"api_base"`
	APIKey       string `yaml:"api_key"`
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

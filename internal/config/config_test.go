package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigStructYAMLTags(t *testing.T) {
	src := `profiles:
  - name: openai-official
    provider: openai
    api_base: https://api.openai.com/v1
    api_key: sk-xxxx
    models:
      - gpt-4o
default_profile: openai-official
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	p := cfg.Profiles[0]
	if p.Name != "openai-official" {
		t.Errorf("Name = %q, want %q", p.Name, "openai-official")
	}
	if p.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", p.Provider, "openai")
	}
	if p.APIBase != "https://api.openai.com/v1" {
		t.Errorf("APIBase = %q, want %q", p.APIBase, "https://api.openai.com/v1")
	}
	if p.APIKey != "sk-xxxx" {
		t.Errorf("APIKey = %q, want %q", p.APIKey, "sk-xxxx")
	}
	if len(p.Models) != 1 || p.Models[0].Name != "gpt-4o" {
		t.Errorf("Models = %v, want [gpt-4o]", p.Models)
	}
	if p.Models[0].ContextWindowTokens != nil || p.Models[0].MaxOutputTokens != nil {
		t.Errorf("scalar model should have no token limits, got %+v", p.Models[0])
	}
	if cfg.DefaultProfile != "openai-official" {
		t.Errorf("DefaultProfile = %q, want %q", cfg.DefaultProfile, "openai-official")
	}
}

// TestLoad_ScalarModelsNormalizeToNameOnly 驗證既有 scalar models 清單載入後
// 正規化為只有 Name、無 token 限制的 Model。
func TestLoad_ScalarModelsNormalizeToNameOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte("profiles:\n  - name: p\n    api_base: https://x\n    models: [gpt-4o, gpt-4o-mini]\ndefault_profile: p\n")
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	models := cfg.Profiles[0].Models
	if len(models) != 2 || models[0].Name != "gpt-4o" || models[1].Name != "gpt-4o-mini" {
		t.Fatalf("Models = %+v, want names [gpt-4o gpt-4o-mini]", models)
	}
	for i, m := range models {
		if m.ContextWindowTokens != nil || m.MaxOutputTokens != nil {
			t.Errorf("models[%d] should have nil token limits, got %+v", i, m)
		}
	}
}

// TestLoad_MappingModelLimitsLoad 驗證 mapping 形式的模型項目可載入
// name、context_window_tokens 與 max_output_tokens。
func TestLoad_MappingModelLimitsLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte(`profiles:
  - name: p
    api_base: https://x
    models:
      - name: gpt-5.4
        context_window_tokens: 1000000
        max_output_tokens: 128000
default_profile: p
`)
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	models := cfg.Profiles[0].Models
	if len(models) != 1 || models[0].Name != "gpt-5.4" {
		t.Fatalf("Models = %+v, want one model named gpt-5.4", models)
	}
	if models[0].ContextWindowTokens == nil || *models[0].ContextWindowTokens != 1000000 {
		t.Errorf("ContextWindowTokens = %v, want 1000000", models[0].ContextWindowTokens)
	}
	if models[0].MaxOutputTokens == nil || *models[0].MaxOutputTokens != 128000 {
		t.Errorf("MaxOutputTokens = %v, want 128000", models[0].MaxOutputTokens)
	}
}

// TestLoad_MappingModelPartialLimits 驗證 mapping 只設定部分欄位時，
// 未設定的欄位為 nil（無值）。
func TestLoad_MappingModelPartialLimits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte(`profiles:
  - name: p
    api_base: https://x
    models:
      - name: gpt-5.4-mini
        max_output_tokens: 32768
default_profile: p
`)
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	m := cfg.Profiles[0].Models[0]
	if m.Name != "gpt-5.4-mini" {
		t.Fatalf("Name = %q, want gpt-5.4-mini", m.Name)
	}
	if m.ContextWindowTokens != nil {
		t.Errorf("ContextWindowTokens = %v, want nil (unset)", m.ContextWindowTokens)
	}
	if m.MaxOutputTokens == nil || *m.MaxOutputTokens != 32768 {
		t.Errorf("MaxOutputTokens = %v, want 32768", m.MaxOutputTokens)
	}
}

// TestLoad_ProfileDefaultModelLimitsLoad 驗證 profile 的 default_model_limits
// 載入後保留為 profile 級預設，不套用到個別模型物件。
func TestLoad_ProfileDefaultModelLimitsLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte(`profiles:
  - name: p
    api_base: https://x
    default_model_limits:
      context_window_tokens: 272000
      max_output_tokens: 16384
    models:
      - gpt-4o
default_profile: p
`)
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	p := cfg.Profiles[0]
	if p.DefaultModelLimits == nil {
		t.Fatal("DefaultModelLimits = nil, want non-nil")
	}
	if p.DefaultModelLimits.ContextWindowTokens == nil || *p.DefaultModelLimits.ContextWindowTokens != 272000 {
		t.Errorf("default ContextWindowTokens = %v, want 272000", p.DefaultModelLimits.ContextWindowTokens)
	}
	if p.DefaultModelLimits.MaxOutputTokens == nil || *p.DefaultModelLimits.MaxOutputTokens != 16384 {
		t.Errorf("default MaxOutputTokens = %v, want 16384", p.DefaultModelLimits.MaxOutputTokens)
	}
	if len(p.Models) != 1 || p.Models[0].ContextWindowTokens != nil {
		t.Errorf("profile defaults must not apply to stored models, got %+v", p.Models[0])
	}
}

// TestLoad_MixedScalarAndMappingModels 驗證同一清單可混用 scalar 與 mapping。
func TestLoad_MixedScalarAndMappingModels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte(`profiles:
  - name: p
    api_base: https://x
    models:
      - gpt-4o
      - name: gpt-5.4
        context_window_tokens: 1000000
default_profile: p
`)
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	models := cfg.Profiles[0].Models
	if len(models) != 2 || models[0].Name != "gpt-4o" || models[1].Name != "gpt-5.4" {
		t.Fatalf("Models = %+v, want [gpt-4o gpt-5.4]", models)
	}
	if models[1].ContextWindowTokens == nil || *models[1].ContextWindowTokens != 1000000 {
		t.Errorf("models[1].ContextWindowTokens = %v, want 1000000", models[1].ContextWindowTokens)
	}
}

// TestLoad_LegacyDefaultModelMigrated 驗證含舊 default_model 欄位且無 models
// 清單的設定檔載入後，default_model 遷移為單元素 models 清單。
func TestLoad_LegacyDefaultModelMigrated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte("profiles:\n  - name: openai-official\n    provider: openai\n    api_base: https://api.openai.com/v1\n    api_key: sk-xxxx\n    default_model: gpt-4o\ndefault_profile: openai-official\n")
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	p := cfg.Profiles[0]
	if len(p.Models) != 1 || p.Models[0].Name != "gpt-4o" {
		t.Errorf("Models = %v, want [gpt-4o] (legacy default_model migrated)", p.Models)
	}
}

// TestLoad_ModelsPreservedOverLegacy 驗證已含 models 清單的 profile 不被
// 舊 default_model 欄位覆寫。
func TestLoad_ModelsPreservedOverLegacy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte("profiles:\n  - name: p\n    provider: openai\n    api_base: https://x\n    default_model: legacy\n    models:\n      - a\n      - b\ndefault_profile: p\n")
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Profiles[0].Models) != 2 || cfg.Profiles[0].Models[0].Name != "a" || cfg.Profiles[0].Models[1].Name != "b" {
		t.Errorf("Models = %v, want [a b] (existing models must not be overwritten by legacy default_model)", cfg.Profiles[0].Models)
	}
}

// TestSave_OmitsLegacyDefaultModel 驗證儲存後的檔案不含 default_model 欄位，
// 僅含 models 清單。
func TestSave_OmitsLegacyDefaultModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &Config{
		Profiles: []Profile{
			{Name: "p", Provider: "openai", APIBase: "https://x", Models: []Model{{Name: "gpt-4o"}}},
		},
		DefaultProfile: "p",
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if strings.Contains(string(data), "default_model") {
		t.Errorf("saved file still contains legacy default_model field:\n%s", string(data))
	}
	if !strings.Contains(string(data), "models:") {
		t.Errorf("saved file missing models field:\n%s", string(data))
	}
}

// TestSave_WritesMappingModels 驗證儲存時模型以 mapping 形式寫出（含 name 欄位），
// 未設定的 token 欄位不寫出；default_model_limits 僅在非 nil 時輸出。
func TestSave_WritesMappingModels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	ctx := int64(1000000)
	cfg := &Config{
		Profiles: []Profile{
			{
				Name:     "p",
				Provider: "openai",
				APIBase:  "https://x",
				Models: []Model{
					{Name: "gpt-5.4", ContextWindowTokens: &ctx},
					{Name: "gpt-4o"},
				},
				DefaultModelLimits: &ModelLimits{MaxOutputTokens: ptrInt64(16384)},
			},
		},
		DefaultProfile: "p",
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	out := string(data)
	for _, want := range []string{"name: gpt-5.4", "context_window_tokens: 1000000", "name: gpt-4o", "default_model_limits:", "max_output_tokens: 16384"} {
		if !strings.Contains(out, want) {
			t.Errorf("saved file missing %q:\n%s", want, out)
		}
	}
	// 未設定 token 的模型不應寫出 token 欄位。
	if strings.Contains(out, "max_output_tokens: null") || strings.Contains(out, "context_window_tokens: null") {
		t.Errorf("saved file should omit unset token fields:\n%s", out)
	}
	// 寫回的檔案可再載入且保留同樣欄位。
	cfg2, err := Load(path)
	if err != nil {
		t.Fatalf("roundtrip Load failed: %v", err)
	}
	m := cfg2.Profiles[0].Models
	if len(m) != 2 || m[0].Name != "gpt-5.4" || m[0].ContextWindowTokens == nil || *m[0].ContextWindowTokens != 1000000 {
		t.Errorf("roundtrip Models = %+v, want gpt-5.4 with 1000000", m)
	}
	if cfg2.Profiles[0].DefaultModelLimits == nil || cfg2.Profiles[0].DefaultModelLimits.MaxOutputTokens == nil || *cfg2.Profiles[0].DefaultModelLimits.MaxOutputTokens != 16384 {
		t.Errorf("roundtrip DefaultModelLimits = %+v, want max_output_tokens 16384", cfg2.Profiles[0].DefaultModelLimits)
	}
}

// ptrInt64 回傳 v 的指標，供測試建構 Model 使用。
func ptrInt64(v int64) *int64 { return &v }

func TestLoad_DuplicateModelNamesRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte(`profiles:
  - name: p
    api_base: https://x
    models:
      - gpt-4o
      - name: gpt-4o
        context_window_tokens: 1000
default_profile: p
`)
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for duplicate model names, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "p") || !strings.Contains(msg, "gpt-4o") {
		t.Errorf("error should identify profile p and duplicate name gpt-4o, got: %s", msg)
	}
}

func TestLoad_EmptyModelNameRejected(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte(`profiles:
  - name: p
    api_base: https://x
    models:
      - name: ""
default_profile: p
`)
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for empty model name, got nil")
	}
	if !strings.Contains(err.Error(), "p") {
		t.Errorf("error should identify profile p, got: %s", err.Error())
	}
}

// TestLoad_NonPositiveModelTokensRejected 驗證模型欄位中零或負數 token 值
// 會使載入失敗，錯誤指出 profile、模型與欄位。
func TestLoad_NonPositiveModelTokensRejected(t *testing.T) {
	cases := []struct {
		field string
		value string
	}{
		{"context_window_tokens", "0"},
		{"context_window_tokens", "-1000"},
		{"max_output_tokens", "0"},
		{"max_output_tokens", "-1"},
	}
	for _, tc := range cases {
		t.Run(tc.field+"="+tc.value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			src := []byte("profiles:\n  - name: p\n    api_base: https://x\n    models:\n      - name: gpt-5.4\n        " + tc.field + ": " + tc.value + "\ndefault_profile: p\n")
			if err := os.WriteFile(path, src, 0600); err != nil {
				t.Fatalf("setup write failed: %v", err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatalf("expected error for %s = %s, got nil", tc.field, tc.value)
			}
			msg := err.Error()
			for _, want := range []string{"p", "gpt-5.4", tc.field} {
				if !strings.Contains(msg, want) {
					t.Errorf("error %q should identify %q", msg, want)
				}
			}
		})
	}
}

// TestLoad_NonPositiveDefaultTokensRejected 驗證 default_model_limits 的
// 零或負值會使載入失敗，錯誤指出 profile 與欄位。
func TestLoad_NonPositiveDefaultTokensRejected(t *testing.T) {
	cases := []struct {
		field string
		value string
	}{
		{"context_window_tokens", "0"},
		{"context_window_tokens", "-272000"},
		{"max_output_tokens", "0"},
		{"max_output_tokens", "-16384"},
	}
	for _, tc := range cases {
		t.Run(tc.field+"="+tc.value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			src := []byte("profiles:\n  - name: p\n    api_base: https://x\n    default_model_limits:\n      " + tc.field + ": " + tc.value + "\n    models: [gpt-4o]\ndefault_profile: p\n")
			if err := os.WriteFile(path, src, 0600); err != nil {
				t.Fatalf("setup write failed: %v", err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatalf("expected error for default %s = %s, got nil", tc.field, tc.value)
			}
			msg := err.Error()
			if !strings.Contains(msg, "p") || !strings.Contains(msg, tc.field) {
				t.Errorf("error %q should identify profile p and field %q", msg, tc.field)
			}
		})
	}
}

// TestLoad_NoCrossFieldOrderingValidation 驗證 context 與 output 任何正數
// 排序皆可載入（不做跨欄位大小驗證）。
func TestLoad_NoCrossFieldOrderingValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	src := []byte(`profiles:
  - name: p
    api_base: https://x
    models:
      - name: mixed
        context_window_tokens: 16384
        max_output_tokens: 1000000
default_profile: p
`)
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load should accept any positive context/output order, got error: %v", err)
	}
	m := cfg.Profiles[0].Models[0]
	if m.MaxOutputTokens == nil || *m.MaxOutputTokens != 1000000 {
		t.Errorf("MaxOutputTokens = %v, want 1000000", m.MaxOutputTokens)
	}
}

// TestProfile_ModelNamesOmitsTokenMetadata 驗證 ModelNames 只回傳名稱。
func TestProfile_ModelNamesOmitsTokenMetadata(t *testing.T) {
	p := &Profile{
		Name: "p",
		Models: []Model{
			{Name: "gpt-4o"},
			{Name: "gpt-5.4", ContextWindowTokens: ptrInt64(1000000), MaxOutputTokens: ptrInt64(128000)},
		},
	}
	got := p.ModelNames()
	if len(got) != 2 || got[0] != "gpt-4o" || got[1] != "gpt-5.4" {
		t.Errorf("ModelNames = %v, want [gpt-4o gpt-5.4]", got)
	}
}

// errContainsPath 回報 err 是否提及 path，比對時使用正規化
// （正斜線）路徑，避免 Go 在 Windows 上以 %q 反斜線跳脫造成誤判。
func errContainsPath(t *testing.T, err error, path string) bool {
	t.Helper()
	got := filepath.ToSlash(err.Error())
	want := filepath.ToSlash(path)
	return strings.Contains(got, want)
}

func TestLoad_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "config.yaml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !errContainsPath(t, err, path) {
		t.Errorf("error %q does not contain path %q", err.Error(), path)
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	bad := []byte("profiles:\n  - name: x\n    broken: [unbalanced")
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatalf("setup write failed: %v", err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
	if !errContainsPath(t, err, path) {
		t.Errorf("error %q does not contain path %q", err.Error(), path)
	}
}

func TestSave_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".byok", "config.yaml")
	cfg := &Config{
		Profiles: []Profile{
			{
				Name:     "openai-official",
				APIKey:   "sk-xxxx",
				Provider: "openai",
			},
		},
		DefaultProfile: "openai-official",
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(loaded.Profiles))
	}
	if loaded.Profiles[0].Name != "openai-official" {
		t.Errorf("Name = %q, want %q", loaded.Profiles[0].Name, "openai-official")
	}
	if loaded.Profiles[0].APIKey != "sk-xxxx" {
		t.Errorf("APIKey = %q, want %q", loaded.Profiles[0].APIKey, "sk-xxxx")
	}
	if loaded.DefaultProfile != "openai-official" {
		t.Errorf("DefaultProfile = %q, want %q", loaded.DefaultProfile, "openai-official")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath failed: %v", err)
	}
	want := filepath.Join(".byok", "config.yaml")
	if !strings.HasSuffix(path, want) {
		t.Errorf("path %q does not end with %q", path, want)
	}
}

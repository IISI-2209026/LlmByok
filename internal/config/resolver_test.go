package config

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

// fakeResolver 用於測試時替換全域 Resolver。
type fakeResolver struct {
	key    string
	source Source
	err    error
}

func (f fakeResolver) Resolve(p Profile) (string, Source, error) {
	return f.key, f.source, f.err
}

func TestResolveFromKeychain(t *testing.T) {
	keyring.MockInit()
	if err := keyring.Set("byok", "profile:test", "secret-key"); err != nil {
		t.Fatalf("setup keyring: %v", err)
	}

	r := DefaultResolver{}
	key, src, err := r.Resolve(Profile{Name: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "secret-key" {
		t.Errorf("key = %q, want %q", key, "secret-key")
	}
	if src != SourceKeychain {
		t.Errorf("source = %v, want SourceKeychain", src)
	}
}

func TestResolveFromPlaintext(t *testing.T) {
	keyring.MockInit() // empty mock; no keychain entry for "fallback"

	r := DefaultResolver{}
	key, src, err := r.Resolve(Profile{Name: "fallback", APIKey: "plain-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "plain-key" {
		t.Errorf("key = %q, want %q", key, "plain-key")
	}
	if src != SourcePlaintext {
		t.Errorf("source = %v, want SourcePlaintext", src)
	}
}

func TestResolveMissing(t *testing.T) {
	keyring.MockInit()

	r := DefaultResolver{}
	_, src, err := r.Resolve(Profile{Name: "nokey"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if src != SourceMissing {
		t.Errorf("source = %v, want SourceMissing", src)
	}
}

func TestResolveKeychainTakesPrecedence(t *testing.T) {
	keyring.MockInit()
	if err := keyring.Set("byok", "profile:both", "chain-key"); err != nil {
		t.Fatalf("setup keyring: %v", err)
	}

	r := DefaultResolver{}
	key, src, err := r.Resolve(Profile{Name: "both", APIKey: "plain-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "chain-key" {
		t.Errorf("key = %q, want %q (keychain should take precedence)", key, "chain-key")
	}
	if src != SourceKeychain {
		t.Errorf("source = %v, want SourceKeychain", src)
	}
}

func TestSourceString(t *testing.T) {
	tests := []struct {
		src  Source
		want string
	}{
		{SourceKeychain, "keychain"},
		{SourcePlaintext, "plaintext"},
		{SourceMissing, "missing"},
	}
	for _, tc := range tests {
		if got := tc.src.String(); got != tc.want {
			t.Errorf("%d.String() = %q, want %q", tc.src, got, tc.want)
		}
	}
}

// 確認 KeyResolver 介面可被 fake 替換。
func TestResolverInterfaceReplaceable(t *testing.T) {
	orig := Resolver
	t.Cleanup(func() { Resolver = orig })

	Resolver = fakeResolver{key: "fake", source: SourceKeychain, err: nil}

	key, src, err := Resolver.Resolve(Profile{Name: "any"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key != "fake" {
		t.Errorf("key = %q, want %q", key, "fake")
	}
	if src != SourceKeychain {
		t.Errorf("source = %v, want SourceKeychain", src)
	}
}

func TestResolverErrorPropagation(t *testing.T) {
	orig := Resolver
	t.Cleanup(func() { Resolver = orig })

	wantErr := errors.New("custom resolver error")
	Resolver = fakeResolver{err: wantErr}

	_, _, err := Resolver.Resolve(Profile{Name: "any"})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
}
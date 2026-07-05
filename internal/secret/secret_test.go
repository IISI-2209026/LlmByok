package secret

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestStoreAndLoad(t *testing.T) {
	keyring.MockInit()
	if err := Store("openai-official", "sk-xxxx"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	got, err := Load("openai-official")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != "sk-xxxx" {
		t.Errorf("Load = %q, want sk-xxxx", got)
	}
}

func TestStoreOverwrite(t *testing.T) {
	keyring.MockInit()
	if err := Store("openai-official", "sk-old"); err != nil {
		t.Fatalf("Store old: %v", err)
	}
	if err := Store("openai-official", "sk-new"); err != nil {
		t.Fatalf("Store new: %v", err)
	}
	got, err := Load("openai-official")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != "sk-new" {
		t.Errorf("Load = %q, want sk-new", got)
	}
}

func TestDeleteThenExists(t *testing.T) {
	keyring.MockInit()
	if err := Store("openai-official", "sk-xxxx"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if err := Delete("openai-official"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	exists, err := Exists("openai-official")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Errorf("Exists = true after Delete, want false")
	}
	_, err = Load("openai-official")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load after Delete: err = %v, want ErrNotFound", err)
	}
}

func TestLoadNotFound(t *testing.T) {
	keyring.MockInit()
	_, err := Load("never-stored")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load non-existent: err = %v, want ErrNotFound", err)
	}
}

func TestBackendUnavailable(t *testing.T) {
	keyring.MockInitWithError(errors.New("dbus: service not running"))
	_, err := Load("openai-official")
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Errorf("Load with backend down: err = %v, want ErrBackendUnavailable", err)
	}
	err = Store("openai-official", "sk-xxxx")
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Errorf("Store with backend down: err = %v, want ErrBackendUnavailable", err)
	}
	_, err = Exists("openai-official")
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Errorf("Exists with backend down: err = %v, want ErrBackendUnavailable", err)
	}
}
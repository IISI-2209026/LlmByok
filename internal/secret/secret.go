// Package secret 封裝 OS keychain 操作，提供跨平台的 API 金鑰儲存。
//
// 採用 github.com/zalando/go-keyring 作為 keychain 抽象層。
// service 固定為 "byok"，key 為 "profile:<profileName>"。
package secret

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// serviceName 為 keychain 中的 service 名稱。
	serviceName = "byok"
	// keyPrefix 為每個 profile key 的前綴。
	keyPrefix = "profile:"
)

// ErrNotFound 表示 keychain 中找不到該 profile 的金鑰。
var ErrNotFound = errors.New("secret: key not found in keychain")

// ErrBackendUnavailable 表示 OS keychain backend 不可用
// （例如 headless Linux 無 secret-service daemon）。
var ErrBackendUnavailable = errors.New("secret: keychain backend unavailable")

// keyName 回傳 profile 在 keychain 中的 key 名稱。
func keyName(profileName string) string {
	return keyPrefix + profileName
}

// Store 將 apiKey 存入 keychain 的 profile:<profileName> 位置。
// 已存在值會被覆寫。
func Store(profileName, apiKey string) error {
	if err := keyring.Set(serviceName, keyName(profileName), apiKey); err != nil {
		return fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
	}
	return nil
}

// Load 自 keychain 讀取 profile:<profileName> 的金鑰。
// 找不到時回傳 ErrNotFound；backend 不可用時回傳 ErrBackendUnavailable。
func Load(profileName string) (string, error) {
	val, err := keyring.Get(serviceName, keyName(profileName))
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
	}
	return val, nil
}

// Delete 自 keychain 刪除 profile:<profileName> 的金鑰。
// 找不到時回傳 ErrNotFound。
func Delete(profileName string) error {
	if err := keyring.Delete(serviceName, keyName(profileName)); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
	}
	return nil
}

// Exists 檢查 keychain 中是否存在 profile:<profileName> 的金鑰。
// backend 不可用時回傳 false 與 ErrBackendUnavailable。
func Exists(profileName string) (bool, error) {
	_, err := keyring.Get(serviceName, keyName(profileName))
	if err == nil {
		return true, nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return false, nil
	}
	return false, fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
}
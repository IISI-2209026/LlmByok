package secret

// 本檔案提供測試用的金鑰函式注入介面，僅供測試程式碼使用。
// 透過 StoreFnForTest / RestoreFnsForTest / SetStoreFnForTest 可在測試中
// 替換 keychain 操作以模擬部分失敗等情境。

// StoreFunc 是 keychain Store 操作的函式型別。
type StoreFunc func(service, user, password string) error

// LoadFunc 是 keychain Load 操作的函式型別。
type LoadFunc func(service, user string) (string, error)

// DeleteFunc 是 keychain Delete 操作的函式型別。
type DeleteFunc func(service, user string) error

// StoreFnForTest 回傳目前的 store/load/delete 函式，供測試還原。
func StoreFnForTest() (StoreFunc, LoadFunc, DeleteFunc) {
	return storeFn, loadFn, deleteFn
}

// RestoreFnsForTest 還原 store/load/delete 函式至測試前的狀態。
func RestoreFnsForTest(s StoreFunc, l LoadFunc, d DeleteFunc) {
	storeFn = s
	loadFn = l
	deleteFn = d
}

// SetStoreFnForTest 替換 store 函式以模擬特定失敗情境。
func SetStoreFnForTest(fn StoreFunc) {
	storeFn = fn
}
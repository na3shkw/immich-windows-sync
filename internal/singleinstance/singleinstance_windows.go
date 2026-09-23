package singleinstance

import (
	"errors"

	"golang.org/x/sys/windows"
)

// 取得したハンドルはCloseしない。Closeすると別プロセスからは「誰も掴んでいないMutex」に見えてしまい、
// 多重起動チェックの意味がなくなる（プロセス終了時にOSが自動的に解放する）。
func createWindowsMutex(name string) (alreadyRunning bool, err error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return false, err
	}
	_, err = windows.CreateMutex(nil, false, namePtr)
	if err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

package startup

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const startupRunKeyPath = "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Run"

type Startup struct {
	keyName string
}

func NewStartup(isDev bool) *Startup {
	keyName := "ImmichWindowsSync"
	if isDev {
		keyName += "-dev"
	}

	return &Startup{
		keyName: keyName,
	}
}

func (s *Startup) Register() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRunKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	err = key.SetStringValue(s.keyName, exePath)
	if err != nil {
		return err
	}
	return nil
}

func (s *Startup) UnRegister() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRunKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	err = key.DeleteValue(s.keyName)
	if err != nil {
		return err
	}
	return nil
}

func (s *Startup) IsRegistered() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRunKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()
	_, _, err = key.GetStringValue(s.keyName)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

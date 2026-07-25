package startup

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

const startupRunKeyPath = "SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Run"

// registryKey は registry.Key のうち Startup が使うメソッドだけを切り出したインターフェース。
// テストでは実際のレジストリに触れないフェイク実装に差し替える。
type registryKey interface {
	SetStringValue(name, value string) error
	GetStringValue(name string) (val string, valtype uint32, err error)
	DeleteValue(name string) error
	Close() error
}

type openKeyFunc func(access uint32) (registryKey, error)

type Startup struct {
	keyName string
	openKey openKeyFunc
}

func NewStartup(isDev bool) *Startup {
	keyName := "ImmichWindowsSync"
	if isDev {
		keyName += "-dev"
	}

	return &Startup{
		keyName: keyName,
		openKey: func(access uint32) (registryKey, error) {
			return registry.OpenKey(registry.CURRENT_USER, startupRunKeyPath, access)
		},
	}
}

func (s *Startup) Register() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	key, err := s.openKey(registry.SET_VALUE)
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
	key, err := s.openKey(registry.SET_VALUE)
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
	key, err := s.openKey(registry.QUERY_VALUE)
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

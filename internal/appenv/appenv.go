package appenv

import (
	"fmt"
	"os"
	"path/filepath"
)

func getDir(name string) (string, error) {
	val := os.Getenv(name)
	if val == "" {
		return "", fmt.Errorf(`Environment variable "%s" is empty.`, name)
	}
	path := filepath.Join(val, appDirName)
	return path, nil
}

func AppDir() (string, error) {
	return getDir("APPDATA")
}

func LocalAppDir() (string, error) {
	return getDir("LOCALAPPDATA")
}

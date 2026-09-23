package appenv

import (
	"fmt"
	"os"
	"path/filepath"
)

func AppDir() (string, error) {
	appdataDir := os.Getenv("APPDATA")
	if appdataDir == "" {
		return "", fmt.Errorf(`Environment variable "APPDATA" is empty.`)
	}
	path := filepath.Join(appdataDir, appDirName)
	return path, nil
}

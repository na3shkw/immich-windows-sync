package appenv

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppDir_Error(t *testing.T) {
	t.Setenv("APPDATA", "")
	_, err := AppDir()
	assert.Error(t, err)
}

func TestAppDir(t *testing.T) {
	t.Setenv("APPDATA", "C:\\Users\\test\\AppData\\Roaming")
	appDir, err := AppDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join("C:\\Users\\test\\AppData\\Roaming", appDirName), appDir)
}

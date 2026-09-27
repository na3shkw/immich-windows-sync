package startup

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

type fakeRegistryKey struct {
	registryValues map[string]string
}

func newFakeRegistryKey() *fakeRegistryKey {
	return &fakeRegistryKey{
		registryValues: map[string]string{},
	}
}

func (f *fakeRegistryKey) SetStringValue(name, value string) error {
	f.registryValues[name] = value
	return nil
}

func (f *fakeRegistryKey) GetStringValue(name string) (string, uint32, error) {
	value, ok := f.registryValues[name]
	if ok {
		return value, 0, nil
	}
	return "", 0, registry.ErrNotExist
}

func (f *fakeRegistryKey) DeleteValue(name string) error {
	delete(f.registryValues, name)
	return nil
}

func (f *fakeRegistryKey) Close() error {
	return nil
}

func newTestStartup(t *testing.T) *Startup {
	t.Helper()
	s, _ := newTestStartupWithFake(t)
	return s
}

func newTestStartupWithFake(t *testing.T) (*Startup, *fakeRegistryKey) {
	t.Helper()
	fake := newFakeRegistryKey()
	return &Startup{
		keyName: "ImmichWindowsSync-test",
		openKey: func(access uint32) (registryKey, error) {
			return fake, nil
		},
	}, fake
}

func TestStartup_RegisterAndIsRegistered(t *testing.T) {
	s := newTestStartup(t)

	registered, err := s.IsRegistered()
	require.NoError(t, err)
	assert.False(t, registered)

	require.NoError(t, s.Register())

	registered, err = s.IsRegistered()
	require.NoError(t, err)
	assert.True(t, registered)
}

func TestStartup_UnRegister(t *testing.T) {
	s := newTestStartup(t)
	require.NoError(t, s.Register())

	require.NoError(t, s.UnRegister())

	registered, err := s.IsRegistered()
	require.NoError(t, err)
	assert.False(t, registered)
}

func TestStartup_Register(t *testing.T) {
	exePath, err := os.Executable()
	require.NoError(t, err)
	want := `"` + exePath + `" --hidden`

	tests := []struct {
		name     string
		existing map[string]string
	}{
		{
			name:     "未登録なら新規に登録する",
			existing: map[string]string{},
		},
		{
			name:     "以前の形式（--hidden なし）で登録済みなら上書きする",
			existing: map[string]string{"ImmichWindowsSync-test": exePath},
		},
		{
			name:     "他のアプリの登録値には触れない",
			existing: map[string]string{"OtherApp": `"C:\Other\other.exe"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, fake := newTestStartupWithFake(t)
			for name, value := range tt.existing {
				fake.registryValues[name] = value
			}

			require.NoError(t, s.Register())

			assert.Equal(t, want, fake.registryValues[s.keyName])
			for name, value := range tt.existing {
				if name != s.keyName {
					assert.Equal(t, value, fake.registryValues[name])
				}
			}
		})
	}
}

package startup

import (
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
	fake := newFakeRegistryKey()
	return &Startup{
		keyName: "ImmichWindowsSync-test",
		openKey: func(access uint32) (registryKey, error) {
			return fake, nil
		},
	}
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

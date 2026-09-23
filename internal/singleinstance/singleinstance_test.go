package singleinstance

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	lock := New()
	assert.Equal(t, mutexName, lock.name)
}

func TestLock_Acquire(t *testing.T) {
	t.Run("初回起動時はalreadyRunningがfalseになる", func(t *testing.T) {
		lock := &Lock{
			name: mutexName,
			createMutex: func(name string) (bool, error) {
				assert.Equal(t, mutexName, name)
				return false, nil
			},
		}

		alreadyRunning, err := lock.Acquire()

		require.NoError(t, err)
		assert.False(t, alreadyRunning)
	})

	t.Run("既に起動中の場合はalreadyRunningがtrueになる", func(t *testing.T) {
		lock := &Lock{
			name: mutexName,
			createMutex: func(name string) (bool, error) {
				return true, nil
			},
		}

		alreadyRunning, err := lock.Acquire()

		require.NoError(t, err)
		assert.True(t, alreadyRunning)
	})

	t.Run("Mutex作成に失敗した場合はエラーを返す", func(t *testing.T) {
		wantErr := errors.New("mutex作成エラー")
		lock := &Lock{
			name: mutexName,
			createMutex: func(name string) (bool, error) {
				return false, wantErr
			},
		}

		_, err := lock.Acquire()

		assert.ErrorIs(t, err, wantErr)
	})
}

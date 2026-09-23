package debounce

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDebouncer_Trigger(t *testing.T) {
	t.Run("同一キーへの連続呼び出しは1回にまとめられる", func(t *testing.T) {
		var mu sync.Mutex
		var fired []string
		d := New(50*time.Millisecond, func(key string) {
			mu.Lock()
			defer mu.Unlock()
			fired = append(fired, key)
		})

		d.Trigger("a.jpg")
		d.Trigger("a.jpg")
		d.Trigger("a.jpg")

		time.Sleep(200 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"a.jpg"}, fired)
	})

	t.Run("十分に間隔を空けた呼び出しはそれぞれ発火する", func(t *testing.T) {
		var mu sync.Mutex
		var fired []string
		d := New(30*time.Millisecond, func(key string) {
			mu.Lock()
			defer mu.Unlock()
			fired = append(fired, key)
		})

		d.Trigger("a.jpg")
		time.Sleep(100 * time.Millisecond)
		d.Trigger("a.jpg")
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{"a.jpg", "a.jpg"}, fired)
	})

	t.Run("異なるキーは独立して扱われる", func(t *testing.T) {
		var mu sync.Mutex
		var fired []string
		d := New(30*time.Millisecond, func(key string) {
			mu.Lock()
			defer mu.Unlock()
			fired = append(fired, key)
		})

		d.Trigger("a.jpg")
		d.Trigger("b.jpg")
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		assert.ElementsMatch(t, []string{"a.jpg", "b.jpg"}, fired)
	})
}

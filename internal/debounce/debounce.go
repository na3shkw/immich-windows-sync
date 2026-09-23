// 同一キーに対する短時間の連続呼び出しを1回にまとめる。
package debounce

import (
	"sync"
	"time"
)

// Debouncer は同一キーに対する短時間の連続呼び出しを1回にまとめる。
// ファイルコピーなど、単一の操作からfsnotifyのCreate/Writeが複数回発火することがあり、
// そのたびに同期処理を走らせるのは無駄なため。
type Debouncer struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
	delay  time.Duration
	fire   func(key string)
}

func New(delay time.Duration, fire func(key string)) *Debouncer {
	return &Debouncer{
		timers: make(map[string]*time.Timer),
		delay:  delay,
		fire:   fire,
	}
}

// Trigger は同じkeyに対して短時間に複数回呼ばれても、最後の呼び出しからdelayが経過した
// 時点でfireが1回だけ呼ばれるようにする。d.timersはTriggerの呼び出し元とタイマーの発火
// コールバック（別goroutine）の両方から触られるため、d.muで保護している。
func (d *Debouncer) Trigger(key string) {
	d.mu.Lock()
	val, ok := d.timers[key]
	d.mu.Unlock()
	if ok {
		val.Reset(d.delay)
	} else {
		d.timers[key] = time.AfterFunc(d.delay, func() {
			d.fire(key)
			d.mu.Lock()
			delete(d.timers, key)
			d.mu.Unlock()
		})
	}
}

package main

import (
	"sync"
	"time"
)

// pathDebouncer は同一パスに対する短時間の連続呼び出しを1回にまとめる。
// ファイルコピーなど、単一の操作からfsnotifyのCreate/Writeが複数回発火することがあり、
// そのたびに同期処理を走らせるのは無駄なため。
type pathDebouncer struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
	delay  time.Duration
	fire   func(path string)
}

func newPathDebouncer(delay time.Duration, fire func(path string)) *pathDebouncer {
	return &pathDebouncer{
		timers: make(map[string]*time.Timer),
		delay:  delay,
		fire:   fire,
	}
}

// Trigger は同じpathに対して短時間に複数回呼ばれても、最後の呼び出しからdelayが経過した
// 時点でfireが1回だけ呼ばれるようにする。d.timersはTriggerの呼び出し元とタイマーの発火
// コールバック（別goroutine）の両方から触られるため、d.muで保護している。
func (d *pathDebouncer) Trigger(path string) {
	d.mu.Lock()
	val, ok := d.timers[path]
	d.mu.Unlock()
	if ok {
		val.Reset(d.delay)
	} else {
		d.timers[path] = time.AfterFunc(d.delay, func() {
			d.fire(path)
			d.mu.Lock()
			delete(d.timers, path)
			d.mu.Unlock()
		})
	}
}

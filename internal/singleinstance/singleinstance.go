package singleinstance

const mutexName = `Global\ImmichWindowsSync-SingleInstance`

// createMutexFunc は名前付きMutexの作成を切り出した関数型。
// テストでは実際のWindows APIに触れないフェイク実装に差し替える。
type createMutexFunc func(name string) (alreadyRunning bool, err error)

type Lock struct {
	name        string
	createMutex createMutexFunc
}

func New() *Lock {
	return &Lock{
		name:        mutexName,
		createMutex: createWindowsMutex,
	}
}

// Acquire は多重起動でないかを確認する。
// 既に別プロセスが同名のMutexを保持していれば alreadyRunning=true を返す。
func (l *Lock) Acquire() (alreadyRunning bool, err error) {
	return l.createMutex(l.name)
}

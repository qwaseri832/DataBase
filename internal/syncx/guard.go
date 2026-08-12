package syncx

import "sync"

func Guard(mu sync.Locker, fn func()) {
	if fn == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn()
}

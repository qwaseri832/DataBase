package syncx

import "sync"

// Guard выполняет функцию под мьютексом.
func Guard(mu sync.Locker, fn func()) {
	if fn == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	fn()
}

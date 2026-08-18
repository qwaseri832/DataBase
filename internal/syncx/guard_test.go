package syncx

import (
	"sync"
	"testing"
)

func TestGuardRunsUnderLock(t *testing.T) {
	var (
		mu      sync.Mutex
		counter int
		wg      sync.WaitGroup
	)

	const goroutines = 50
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			Guard(&mu, func() { counter++ })
		}()
	}
	wg.Wait()

	if counter != goroutines {
		t.Errorf("счётчик = %d, ожидалось %d", counter, goroutines)
	}
}

func TestGuardIgnoresNil(t *testing.T) {
	var mu sync.Mutex

	Guard(&mu, nil)

	if !mu.TryLock() {
		t.Fatal("мьютекс остался захваченным")
	}
	mu.Unlock()
}

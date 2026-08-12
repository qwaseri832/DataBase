package storage

import (
	"sync"
	"testing"
)

func TestIDGenNext(t *testing.T) {
	gen := NewIDGen(0)

	for i := int64(1); i <= 10; i++ {
		if got := gen.Next(); got != i {
			t.Errorf("Next() = %v, ожидалось %v", got, i)
		}
	}
}

func TestIDGenStartsAfterLastLSN(t *testing.T) {
	gen := NewIDGen(100)

	if got := gen.Next(); got != 101 {
		t.Errorf("Next() при старте со 100 = %v, ожидалось 101", got)
	}
}

func TestIDGenConcurrentUniqueness(t *testing.T) {
	const (
		goroutines = 16
		perG       = 1000
	)

	gen := NewIDGen(0)
	ids := make([]int64, goroutines*perG)

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perG; i++ {
				ids[g*perG+i] = gen.Next()
			}
		}(g)
	}
	wg.Wait()

	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			t.Fatalf("ID %d выдан дважды", id)
		}
		seen[id] = struct{}{}
	}

	if got := gen.Next(); got != int64(len(ids))+1 {
		t.Errorf("после %d выдач Next() = %d, ожидалось %d", len(ids), got, len(ids)+1)
	}
}

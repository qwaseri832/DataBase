package syncx

import (
	"context"
	"testing"
	"time"
)

func TestSemaphoreBlocksWhenSlotsAreTaken(t *testing.T) {
	s := NewSemaphore(1)

	if !s.AcquireContext(context.Background()) {
		t.Fatal("первый AcquireContext() = false")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if s.AcquireContext(ctx) {
		t.Error("AcquireContext() = true при исчерпанных слотах")
	}

	s.Release()

	if !s.AcquireContext(context.Background()) {
		t.Error("AcquireContext() = false после освобождения слота")
	}
}

func TestSemaphoreUnlimited(t *testing.T) {
	s := NewSemaphore(0)

	for i := 0; i < 100; i++ {
		if !s.AcquireContext(context.Background()) {
			t.Fatalf("AcquireContext() = false на итерации %d", i)
		}
	}

	s.Release()
}

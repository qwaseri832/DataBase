package syncx

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestPromiseResolve(t *testing.T) {
	p := NewPromise[error]()
	want := errors.New("ошибка записи")

	go p.Resolve(want)

	if got := p.Future().Await(); !errors.Is(got, want) {
		t.Errorf("Await() = %v, ожидалось %v", got, want)
	}
}

func TestFutureAwaitIsRepeatable(t *testing.T) {
	p := NewPromise[int]()
	p.Resolve(42)

	f := p.Future()
	for i := 0; i < 3; i++ {
		if got := f.Await(); got != 42 {
			t.Fatalf("вызов %d: Await() = %d, ожидалось 42", i+1, got)
		}
	}
}

func TestPromiseResolveOnceUnderRace(t *testing.T) {
	p := NewPromise[int]()

	var wg sync.WaitGroup
	for i := 1; i <= 8; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			p.Resolve(v)
		}(i)
	}
	wg.Wait()

	got := p.Future().Await()
	if got < 1 || got > 8 {
		t.Errorf("Await() = %d, ожидалось одно из установленных значений", got)
	}

	p.Resolve(999)
	if again := p.Future().Await(); again != got {
		t.Errorf("значение изменилось после повторного Resolve: %d → %d", got, again)
	}
}

func TestFutureAwaitContextCancel(t *testing.T) {
	p := NewPromise[int]()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := p.Future().AwaitContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("AwaitContext() error = %v, ожидалось DeadlineExceeded", err)
	}
}

func TestFutureAwaitContextValue(t *testing.T) {
	p := NewPromise[string]()
	p.Resolve("готово")

	v, err := p.Future().AwaitContext(context.Background())
	if err != nil {
		t.Fatalf("AwaitContext() error = %v", err)
	}
	if v != "готово" {
		t.Errorf("AwaitContext() = %q, ожидалось \"готово\"", v)
	}
}

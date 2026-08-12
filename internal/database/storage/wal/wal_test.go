package wal

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/syncx"
)

type slowFlusher struct {
	delay time.Duration

	mu     sync.Mutex
	writes int
	bytes  int
}

func (f *slowFlusher) Write(data []byte) error {
	time.Sleep(f.delay)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	f.bytes += len(data)
	return nil
}

func (f *slowFlusher) stats() (writes, bytes int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writes, f.bytes
}

func newTestWAL(f Flusher, timeout time.Duration, maxBatch int) *WAL {
	logger := zap.NewNop()
	return New(NewWriter(f, logger), NewReader(emptyScanner{}), timeout, maxBatch)
}

type emptyScanner struct{}

func (emptyScanner) ForEach(func([]byte) error) error { return nil }

func TestAppendDoesNotBlockOnBusyWriter(t *testing.T) {
	flusher := &slowFlusher{delay: 50 * time.Millisecond}
	w := newTestWAL(flusher, time.Hour, 2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	const records = 20
	futures := make([]syncx.Future[error], 0, records)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < records; i++ {
			futures = append(futures, w.Append(int64(i), OpSet, []string{"k", "v"}))
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Append заблокировался: вероятно, отправка в канал идёт под мьютексом")
	}

	for i, f := range futures {
		if err := f.Await(); err != nil {
			t.Fatalf("запись %d: %v", i, err)
		}
	}

	if writes, _ := flusher.stats(); writes != records/2 {
		t.Errorf("записано батчей: %d, ожидалось %d", writes, records/2)
	}
}

func TestAppendFlushesByTimeout(t *testing.T) {
	flusher := &slowFlusher{}
	w := newTestWAL(flusher, 10*time.Millisecond, 1000)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	f := w.Append(1, OpSet, []string{"k", "v"})

	done := make(chan error, 1)
	go func() { done <- f.Await() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Await() = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("батч не сброшен по таймауту")
	}
}

func TestRunDrainsOnShutdown(t *testing.T) {
	flusher := &slowFlusher{}
	w := newTestWAL(flusher, time.Hour, 1000)

	ctx, cancel := context.WithCancel(context.Background())

	stopped := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(stopped)
	}()

	f := w.Append(1, OpSet, []string{"k", "v"})
	cancel()

	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("Run не завершился по отмене контекста")
	}

	if err := f.Await(); err != nil {
		t.Fatalf("Await() = %v", err)
	}
	if writes, _ := flusher.stats(); writes != 1 {
		t.Errorf("записано батчей: %d, ожидался 1", writes)
	}
}

func TestAppendAfterShutdownReturnsErrClosed(t *testing.T) {
	flusher := &slowFlusher{}
	w := newTestWAL(flusher, time.Hour, 1)

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(stopped)
	}()
	cancel()
	<-stopped

	done := make(chan error, 1)
	go func() { done <- w.Append(1, OpSet, []string{"k", "v"}).Await() }()

	select {
	case err := <-done:
		if err != ErrClosed {
			t.Errorf("Await() = %v, ожидалось ErrClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Append повис после остановки WAL")
	}
}

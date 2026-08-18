package wal

import (
	"bytes"
	"context"
	"reflect"
	"sync"
	"sync/atomic"
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
			futures = append(futures, w.Append(OpSet, []string{"k", "v"}))
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

	f := w.Append(OpSet, []string{"k", "v"})

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

	f := w.Append(OpSet, []string{"k", "v"})
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
	go func() { done <- w.Append(OpSet, []string{"k", "v"}).Await() }()

	select {
	case err := <-done:
		if err != ErrClosed {
			t.Errorf("Await() = %v, ожидалось ErrClosed", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Append повис после остановки WAL")
	}
}

type recordingFlusher struct {
	mu   sync.Mutex
	data []byte
}

func (f *recordingFlusher) Write(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = append(f.data, data...)
	return nil
}

func (f *recordingFlusher) records(t *testing.T) []Record {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()

	recs, err := DecodeSegment(f.data)
	if err != nil {
		t.Fatalf("DecodeSegment: %v", err)
	}
	return recs
}

func TestDecodeSegmentReadsWriterOutput(t *testing.T) {
	flusher := &recordingFlusher{}
	w := NewWriter(flusher, zap.NewNop())

	want := []Record{
		{LSN: 1, Op: OpSet, Args: []string{"ключ", "значение"}},
		{LSN: 2, Op: OpDel, Args: []string{"ключ"}},
		{LSN: 3, Op: OpSet, Args: []string{"ещё", "запись"}},
	}

	for _, batch := range [][]Record{want[:2], want[2:]} {
		pendings := make([]Pending, 0, len(batch))
		for _, rec := range batch {
			pendings = append(pendings, newPending(rec.LSN, rec.Op, rec.Args))
		}
		if err := w.Write(pendings); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	got := flusher.records(t)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("прочитано %+v, записывалось %+v", got, want)
	}
}

func TestFlushAppliesInLogOrder(t *testing.T) {
	flusher := &recordingFlusher{}
	w := newTestWAL(flusher, 5*time.Millisecond, 8)

	var (
		mu      sync.Mutex
		applied []int64
	)
	w.OnFlush(func(recs []Record) {
		mu.Lock()
		defer mu.Unlock()
		for _, r := range recs {
			applied = append(applied, r.LSN)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	const (
		writers   = 8
		perWriter = 25
	)

	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perWriter; j++ {
				if err := w.Append(OpSet, []string{"k", "v"}).Await(); err != nil {
					t.Errorf("Append: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	if len(applied) != writers*perWriter {
		t.Fatalf("применено записей: %d, ожидалось %d", len(applied), writers*perWriter)
	}
	for i := 1; i < len(applied); i++ {
		if applied[i] <= applied[i-1] {
			t.Fatalf("порядок применения нарушен: LSN %d после %d", applied[i], applied[i-1])
		}
	}

	written := flusher.records(t)
	if len(written) != len(applied) {
		t.Fatalf("в журнале %d записей, применено %d", len(written), len(applied))
	}
	for i, rec := range written {
		if rec.LSN != applied[i] {
			t.Fatalf("запись %d: в журнале LSN %d, применён %d", i, rec.LSN, applied[i])
		}
	}
}

func TestFlushAppliesBeforeAck(t *testing.T) {
	w := newTestWAL(&recordingFlusher{}, 5*time.Millisecond, 100)

	var applied atomic.Bool
	w.OnFlush(func([]Record) { applied.Store(true) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	if err := w.Append(OpSet, []string{"k", "v"}).Await(); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if !applied.Load() {
		t.Error("Await вернул управление до того, как запись применена к движку")
	}
}

func TestAppendAssignsGrowingLSN(t *testing.T) {
	flusher := &recordingFlusher{}
	w := newTestWAL(flusher, 5*time.Millisecond, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	for i := 0; i < 5; i++ {
		if err := w.Append(OpSet, []string{"k", "v"}).Await(); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	for i, rec := range flusher.records(t) {
		if want := int64(i + 1); rec.LSN != want {
			t.Errorf("запись %d: LSN %d, ожидался %d", i, rec.LSN, want)
		}
	}
}

func TestRecoverContinuesLSN(t *testing.T) {
	flusher := &recordingFlusher{}
	w := New(NewWriter(flusher, zap.NewNop()), NewReader(fixedScanner{
		{LSN: 41, Op: OpSet, Args: []string{"k", "v"}},
		{LSN: 42, Op: OpDel, Args: []string{"k"}},
	}), 5*time.Millisecond, 100)

	recs, err := w.Recover()
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("восстановлено записей: %d, ожидалось 2", len(recs))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	if err := w.Append(OpSet, []string{"k", "v"}).Await(); err != nil {
		t.Fatalf("Append: %v", err)
	}

	written := flusher.records(t)
	if len(written) != 1 || written[0].LSN != 43 {
		t.Errorf("LSN после восстановления: %+v, ожидался 43", written)
	}
}

type fixedScanner []Record

func (s fixedScanner) ForEach(fn func([]byte) error) error {
	var buf bytes.Buffer
	for i := range s {
		if err := s[i].Encode(&buf); err != nil {
			return err
		}
	}
	return fn(buf.Bytes())
}

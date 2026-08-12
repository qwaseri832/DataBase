package storage

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/database/filesystem"
	"github.com/qwaseri832/DataBase/internal/database/storage/engine"
	"github.com/qwaseri832/DataBase/internal/database/storage/wal"
)

func newTestStorage(t *testing.T, opts ...Option) *Storage {
	t.Helper()

	logger := zap.NewNop()
	s, err := New(engine.New(logger, engine.WithPartitions(4)), logger, opts...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestStorageSetGetDel(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	if err := s.Set(ctx, "ключ", "значение"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := s.Get(ctx, "ключ")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "значение" {
		t.Errorf("Get = %q, ожидалось \"значение\"", got)
	}

	if err := s.Del(ctx, "ключ"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if _, err := s.Get(ctx, "ключ"); err != ErrNotFound {
		t.Errorf("Get после Del = %v, ожидалось ErrNotFound", err)
	}
}

func TestGetDoesNotConsumeLSN(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	if err := s.Set(ctx, "k", "v"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	afterFirstWrite := s.idgen.Next()

	for i := 0; i < 100; i++ {
		if _, err := s.Get(ctx, "k"); err != nil {
			t.Fatalf("Get: %v", err)
		}
	}

	if got := s.idgen.Next(); got != afterFirstWrite+1 {
		t.Errorf("после 100 чтений LSN = %d, ожидалось %d", got, afterFirstWrite+1)
	}
}

func TestSlaveIsReadOnly(t *testing.T) {
	s := newTestStorage(t, WithReplica(fakeReplica{master: false}))
	ctx := context.Background()

	if err := s.Set(ctx, "k", "v"); err != ErrReadOnly {
		t.Errorf("Set на реплике = %v, ожидалось ErrReadOnly", err)
	}
	if err := s.Del(ctx, "k"); err != ErrReadOnly {
		t.Errorf("Del на реплике = %v, ожидалось ErrReadOnly", err)
	}
	if _, err := s.Get(ctx, "k"); err != ErrNotFound {
		t.Errorf("Get на реплике = %v, ожидалось ErrNotFound", err)
	}
}

func TestApplyRecordsSurvivesMalformedRecords(t *testing.T) {
	s := newTestStorage(t)

	recs := []wal.Record{
		{LSN: 1, Op: wal.OpSet, Args: []string{"a"}},
		{LSN: 2, Op: wal.OpDel, Args: nil},
		{LSN: 3, Op: wal.Op(99), Args: []string{"x"}},
		{LSN: 4, Op: wal.OpSet, Args: []string{"b", "ок"}},
	}

	last := s.applyRecords(recs)
	if last != 4 {
		t.Errorf("последний LSN = %d, ожидалось 4", last)
	}

	got, err := s.Get(context.Background(), "b")
	if err != nil || got != "ок" {
		t.Errorf("корректная запись не применена: %q, %v", got, err)
	}
}

func TestRecoveryFromWAL(t *testing.T) {
	dir := t.TempDir()
	logger := zap.NewNop()

	newWAL := func() *wal.WAL {
		return wal.New(
			wal.NewWriter(filesystem.NewSegment(dir, 1<<20), logger),
			wal.NewReader(filesystem.NewDirScanner(dir)),
			5*time.Millisecond, 100,
		)
	}

	first := newWAL()
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		first.Run(ctx)
		close(stopped)
	}()

	s1 := newTestStorage(t, WithWAL(first))
	if err := s1.Set(context.Background(), "ключ", "значение"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := s1.Del(context.Background(), "лишний"); err != nil {
		t.Fatalf("Del: %v", err)
	}

	cancel()
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("WAL не остановился")
	}

	s2 := newTestStorage(t, WithWAL(newWAL()))

	got, err := s2.Get(context.Background(), "ключ")
	if err != nil {
		t.Fatalf("после восстановления Get: %v", err)
	}
	if got != "значение" {
		t.Errorf("после восстановления Get = %q, ожидалось \"значение\"", got)
	}

	const recoveredMaxLSN = 2
	if next := s2.idgen.Next(); next <= recoveredMaxLSN {
		t.Errorf("LSN после восстановления = %d, ожидалось больше %d", next, recoveredMaxLSN)
	}
}

type fakeReplica struct{ master bool }

func (r fakeReplica) IsMaster() bool { return r.master }

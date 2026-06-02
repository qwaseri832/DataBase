package wal

import (
	"context"
	"sync"
	"time"

	"spider/internal/syncx"
)

// WAL реализует Write-Ahead Log с батчингом.
type WAL struct {
	writer   *Writer
	reader   *Reader
	timeout  time.Duration
	maxBatch int

	mu      sync.Mutex
	batch   []Pending
	batches chan []Pending
}

func New(w *Writer, r *Reader, timeout time.Duration, maxBatch int) *WAL {
	return &WAL{
		writer:   w,
		reader:   r,
		timeout:  timeout,
		maxBatch: maxBatch,
		batches:  make(chan []Pending, 1),
	}
}

// Start запускает фоновую горутину, которая флашит батчи.
func (w *WAL) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.timeout)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				w.flush()
				return
			case b := <-w.batches:
				w.writer.Write(b)
				ticker.Reset(w.timeout)
			case <-ticker.C:
				w.flush()
			}
		}
	}()
}

// Recover читает все записи из сегментов.
func (w *WAL) Recover() ([]Record, error) {
	return w.reader.ReadAll()
}

// Append добавляет запись в батч; возвращает Future с ошибкой.
func (w *WAL) Append(lsn int64, cmd int, args []string) syncx.Future[error] {
	p := newPending(lsn, cmd, args)

	syncx.Guard(&w.mu, func() {
		w.batch = append(w.batch, p)
		if len(w.batch) >= w.maxBatch {
			w.batches <- w.batch
			w.batch = nil
		}
	})

	return p.Future()
}

func (w *WAL) flush() {
	var b []Pending
	syncx.Guard(&w.mu, func() {
		b = w.batch
		w.batch = nil
	})
	if len(b) > 0 {
		w.writer.Write(b)
	}
}

package wal

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/qwaseri832/DataBase/internal/syncx"
)

var ErrClosed = errors.New("wal: closed")

type WAL struct {
	writer   *Writer
	reader   *Reader
	timeout  time.Duration
	maxBatch int

	mu    sync.Mutex
	batch []Pending

	batches chan []Pending
	done    chan struct{}
	closing sync.Once
}

func New(w *Writer, r *Reader, timeout time.Duration, maxBatch int) *WAL {
	if timeout <= 0 {
		timeout = 10 * time.Millisecond
	}
	if maxBatch <= 0 {
		maxBatch = 100
	}
	return &WAL{
		writer:   w,
		reader:   r,
		timeout:  timeout,
		maxBatch: maxBatch,
		batches:  make(chan []Pending),
		done:     make(chan struct{}),
	}
}

func (w *WAL) Run(ctx context.Context) {
	ticker := time.NewTicker(w.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.close()
			w.drain()
			w.writer.Close()
			return
		case b := <-w.batches:
			w.writer.Write(b)
			ticker.Reset(w.timeout)
		case <-ticker.C:
			w.flush()
		}
	}
}

func (w *WAL) Recover() ([]Record, error) {
	return w.reader.ReadAll()
}

func (w *WAL) Append(lsn int64, op Op, args []string) syncx.Future[error] {
	p := newPending(lsn, op, args)

	var full []Pending
	syncx.Guard(&w.mu, func() {
		w.batch = append(w.batch, p)
		if len(w.batch) >= w.maxBatch {
			full, w.batch = w.batch, nil
		}
	})

	if full != nil {
		w.submit(full)
	}

	return p.Future()
}

func (w *WAL) submit(b []Pending) {
	select {
	case w.batches <- b:
	case <-w.done:
		ack(b, ErrClosed)
	}
}

func (w *WAL) flush() {
	var b []Pending
	syncx.Guard(&w.mu, func() {
		b, w.batch = w.batch, nil
	})
	if len(b) > 0 {
		w.writer.Write(b)
	}
}

func (w *WAL) drain() {
	for {
		select {
		case b := <-w.batches:
			w.writer.Write(b)
		default:
			w.flush()
			return
		}
	}
}

func (w *WAL) close() {
	w.closing.Do(func() { close(w.done) })
}

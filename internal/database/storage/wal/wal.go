package wal

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/qwaseri832/DataBase/internal/syncx"
)

var ErrClosed = errors.New("wal: closed")

type Applier func(recs []Record)

type WAL struct {
	writer   *Writer
	reader   *Reader
	timeout  time.Duration
	maxBatch int

	mu     sync.Mutex
	batch  []Pending
	lsn    int64
	apply  Applier
	closed bool

	full chan struct{}
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
		full:     make(chan struct{}, 1),
	}
}

func (w *WAL) OnFlush(apply Applier) {
	syncx.Guard(&w.mu, func() { w.apply = apply })
}

func (w *WAL) Recover() ([]Record, error) {
	recs, err := w.reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var last int64
	for _, r := range recs {
		if r.LSN > last {
			last = r.LSN
		}
	}

	syncx.Guard(&w.mu, func() { w.lsn = last })
	return recs, nil
}

func (w *WAL) Run(ctx context.Context) {
	ticker := time.NewTicker(w.timeout)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.shutdown()
			return
		case <-w.full:
			w.flush()
			ticker.Reset(w.timeout)
		case <-ticker.C:
			w.flush()
		}
	}
}

func (w *WAL) Append(op Op, args []string) syncx.Future[error] {
	var (
		p      Pending
		closed bool
		full   bool
	)

	syncx.Guard(&w.mu, func() {
		if w.closed {
			closed = true
			return
		}

		w.lsn++
		p = newPending(w.lsn, op, args)
		w.batch = append(w.batch, p)
		full = len(w.batch) >= w.maxBatch
	})

	if closed {
		p = newPending(0, op, args)
		p.Done(ErrClosed)
		return p.Future()
	}

	if full {
		w.wake()
	}

	return p.Future()
}

func (w *WAL) flush() {
	var (
		batch []Pending
		apply Applier
		rest  bool
	)

	syncx.Guard(&w.mu, func() {
		n := min(len(w.batch), w.maxBatch)
		batch, w.batch = w.batch[:n], w.batch[n:]
		apply, rest = w.apply, len(w.batch) > 0
	})

	w.write(batch, apply)

	if rest {
		w.wake()
	}
}

func (w *WAL) shutdown() {
	var (
		batch []Pending
		apply Applier
	)

	syncx.Guard(&w.mu, func() {
		w.closed = true
		batch, w.batch = w.batch, nil
		apply = w.apply
	})

	w.write(batch, apply)
	w.writer.Close()
}

func (w *WAL) write(batch []Pending, apply Applier) {
	if len(batch) == 0 {
		return
	}

	err := w.writer.Write(batch)
	if err == nil && apply != nil {
		apply(records(batch))
	}

	for i := range batch {
		batch[i].Done(err)
	}
}

func (w *WAL) wake() {
	select {
	case w.full <- struct{}{}:
	default:
	}
}

func records(batch []Pending) []Record {
	recs := make([]Record, len(batch))
	for i := range batch {
		recs[i] = batch[i].Record()
	}
	return recs
}

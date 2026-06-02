package wal

import (
	"bytes"

	"go.uber.org/zap"
)

// Flusher умеет записать блок байтов (сегмент на диске).
type Flusher interface {
	Write(data []byte) error
}

// Writer кодирует пачку Pending и пишет через Flusher.
type Writer struct {
	flusher Flusher
	logger  *zap.Logger
}

func NewWriter(f Flusher, l *zap.Logger) *Writer {
	return &Writer{flusher: f, logger: l}
}

func (w *Writer) Write(batch []Pending) {
	var buf bytes.Buffer
	for i := range batch {
		rec := batch[i].Record()
		if err := rec.Encode(&buf); err != nil {
			w.logger.Warn("encode failed", zap.Error(err))
			ack(batch, err)
			return
		}
	}

	err := w.flusher.Write(buf.Bytes())
	if err != nil {
		w.logger.Warn("flush failed", zap.Error(err))
	}
	ack(batch, err)
}

func ack(batch []Pending, err error) {
	for i := range batch {
		batch[i].Done(err)
	}
}

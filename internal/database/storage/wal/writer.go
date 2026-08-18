package wal

import (
	"bytes"
	"io"

	"go.uber.org/zap"
)

type Flusher interface {
	Write(data []byte) error
}

type Writer struct {
	flusher Flusher
	logger  *zap.Logger
}

func NewWriter(f Flusher, l *zap.Logger) *Writer {
	return &Writer{flusher: f, logger: l}
}

func (w *Writer) Write(batch []Pending) error {
	if len(batch) == 0 {
		return nil
	}

	var buf bytes.Buffer
	for i := range batch {
		rec := batch[i].Record()
		if err := rec.Encode(&buf); err != nil {
			w.logger.Warn("encode WAL record", zap.Error(err))
			return err
		}
	}

	if err := w.flusher.Write(buf.Bytes()); err != nil {
		w.logger.Warn("write WAL batch", zap.Error(err))
		return err
	}

	return nil
}

func (w *Writer) Close() {
	c, ok := w.flusher.(io.Closer)
	if !ok {
		return
	}
	if err := c.Close(); err != nil {
		w.logger.Warn("close WAL segment", zap.Error(err))
	}
}

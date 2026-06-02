package replication

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"go.uber.org/zap"

	"spider/internal/database/filesystem"
	"spider/internal/database/storage/wal"
)

// Sender отправляет данные мастеру и возвращает ответ.
type Sender interface {
	Send(data []byte) ([]byte, error)
	Close()
}

// Slave периодически запрашивает новые сегменты у мастера.
type Slave struct {
	sender   Sender
	stream   chan []wal.Record
	interval time.Duration
	walDir   string
	lastSeg  string
	logger   *zap.Logger
}

func NewSlave(s Sender, walDir string, interval time.Duration, l *zap.Logger) *Slave {
	last, _ := filesystem.LastSegment(walDir)
	return &Slave{
		sender:   s,
		stream:   make(chan []wal.Record, 1),
		interval: interval,
		walDir:   walDir,
		lastSeg:  last,
		logger:   l,
	}
}

func (s *Slave) IsMaster() bool                  { return false }
func (s *Slave) Stream() <-chan []wal.Record      { return s.stream }

func (s *Slave) Start(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer func() { t.Stop(); s.sender.Close() }()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.sync()
		}
	}
}

func (s *Slave) sync() {
	req := SyncRequest{AfterSegment: s.lastSeg}
	raw, err := Encode(&req)
	if err != nil {
		s.logger.Error("encode request", zap.Error(err))
		return
	}

	respRaw, err := s.sender.Send(raw)
	if err != nil {
		s.logger.Error("send request", zap.Error(err))
		return
	}

	var resp SyncResponse
	if err := Decode(&resp, respRaw); err != nil {
		s.logger.Error("decode response", zap.Error(err))
		return
	}

	if !resp.OK || resp.SegmentName == "" {
		return
	}

	// Сохраняем сегмент
	path := fmt.Sprintf("%s/%s", s.walDir, resp.SegmentName)
	if err := filesystem.WriteSegment(path, resp.SegmentData); err != nil {
		s.logger.Error("save segment", zap.Error(err))
		return
	}

	// Декодируем и отправляем в stream
	var recs []wal.Record
	buf := bytes.NewBuffer(resp.SegmentData)
	if err := gob.NewDecoder(buf).Decode(&recs); err != nil {
		s.logger.Error("decode records", zap.Error(err))
		return
	}
	s.stream <- recs
	s.lastSeg = resp.SegmentName
}

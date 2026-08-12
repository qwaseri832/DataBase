package replication

import (
	"bytes"
	"context"
	"encoding/gob"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/database/filesystem"
	"github.com/qwaseri832/DataBase/internal/database/storage/wal"
)

type Sender interface {
	Send(data []byte) ([]byte, error)
	Close()
}

type Slave struct {
	sender   Sender
	stream   chan []wal.Record
	interval time.Duration
	walDir   string
	lastSeg  string
	logger   *zap.Logger
}

func NewSlave(s Sender, walDir string, interval time.Duration, l *zap.Logger) *Slave {
	if interval <= 0 {
		interval = time.Second
	}
	last, err := filesystem.LastSegment(walDir)
	if err != nil {
		l.Warn("determine last segment", zap.Error(err))
	}
	return &Slave{
		sender:   s,
		stream:   make(chan []wal.Record, 1),
		interval: interval,
		walDir:   walDir,
		lastSeg:  last,
		logger:   l,
	}
}

func (s *Slave) IsMaster() bool              { return false }
func (s *Slave) Stream() <-chan []wal.Record { return s.stream }

func (s *Slave) Run(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer func() {
		t.Stop()
		s.sender.Close()

		close(s.stream)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.sync(ctx)
		}
	}
}

func (s *Slave) sync(ctx context.Context) {
	req := SyncRequest{AfterSegment: s.lastSeg}
	raw, err := Encode(&req)
	if err != nil {
		s.logger.Error("encode sync request", zap.Error(err))
		return
	}

	respRaw, err := s.sender.Send(raw)
	if err != nil {
		s.logger.Error("send sync request", zap.Error(err))
		return
	}

	var resp SyncResponse
	if err := Decode(&resp, respRaw); err != nil {
		s.logger.Error("decode sync response", zap.Error(err))
		return
	}

	if !resp.OK || resp.SegmentName == "" {
		return
	}

	var recs []wal.Record
	if err := gob.NewDecoder(bytes.NewReader(resp.SegmentData)).Decode(&recs); err != nil {
		s.logger.Error("decode segment", zap.String("segment", resp.SegmentName), zap.Error(err))
		return
	}

	path := filepath.Join(s.walDir, resp.SegmentName)
	if err := filesystem.WriteSegment(path, resp.SegmentData); err != nil {
		s.logger.Error("save segment", zap.String("segment", resp.SegmentName), zap.Error(err))
		return
	}

	select {
	case s.stream <- recs:
	case <-ctx.Done():
		return
	}

	s.lastSeg = resp.SegmentName
	s.logger.Info("segment received from master",
		zap.String("segment", resp.SegmentName),
		zap.Int("records", len(recs)),
	)
}

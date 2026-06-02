package replication

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	"spider/internal/database/filesystem"
)

// Acceptor принимает соединения и вызывает handler для каждого запроса.
type Acceptor interface {
	Serve(ctx context.Context, handler func(ctx context.Context, req []byte) []byte)
}

// Master отдаёт WAL-сегменты по запросу.
type Master struct {
	acceptor Acceptor
	walDir   string
	logger   *zap.Logger
}

func NewMaster(a Acceptor, walDir string, l *zap.Logger) *Master {
	return &Master{acceptor: a, walDir: walDir, logger: l}
}

func (m *Master) IsMaster() bool { return true }

func (m *Master) Start(ctx context.Context) {
	m.acceptor.Serve(ctx, func(_ context.Context, raw []byte) []byte {
		var req SyncRequest
		if err := Decode(&req, raw); err != nil {
			m.logger.Error("decode sync request", zap.Error(err))
			return nil
		}
		data, err := Encode(new(m.handle(req)))
		if err != nil {
			m.logger.Error("encode sync response", zap.Error(err))
		}
		return data
	})
}

func (m *Master) handle(req SyncRequest) SyncResponse {
	next, err := filesystem.NextSegment(m.walDir, req.AfterSegment)
	if err != nil {
		m.logger.Error("find next segment", zap.Error(err))
		return SyncResponse{}
	}
	if next == "" {
		return SyncResponse{OK: true}
	}
	data, err := os.ReadFile(fmt.Sprintf("%s/%s", m.walDir, next))
	if err != nil {
		m.logger.Error("read segment", zap.Error(err))
		return SyncResponse{}
	}
	return SyncResponse{OK: true, SegmentName: next, SegmentData: data}
}

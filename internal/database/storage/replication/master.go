package replication

import (
	"context"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/database/filesystem"
)

type Acceptor interface {
	Serve(ctx context.Context, handler func(ctx context.Context, req []byte) []byte)
}

type Master struct {
	acceptor Acceptor
	walDir   string
	logger   *zap.Logger
}

func NewMaster(a Acceptor, walDir string, l *zap.Logger) *Master {
	return &Master{acceptor: a, walDir: walDir, logger: l}
}

func (m *Master) IsMaster() bool { return true }

func (m *Master) Run(ctx context.Context) {
	m.acceptor.Serve(ctx, func(_ context.Context, raw []byte) []byte {
		var req SyncRequest
		if err := Decode(&req, raw); err != nil {
			m.logger.Error("decode sync request", zap.Error(err))
			return nil
		}

		resp := m.handle(req)
		data, err := Encode(&resp)
		if err != nil {
			m.logger.Error("encode sync response", zap.Error(err))
			return nil
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

	data, err := os.ReadFile(filepath.Join(m.walDir, next))
	if err != nil {
		m.logger.Error("read segment", zap.String("segment", next), zap.Error(err))
		return SyncResponse{}
	}
	return SyncResponse{OK: true, SegmentName: next, SegmentData: data}
}

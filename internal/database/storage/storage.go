package storage

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/qwaseri832/DataBase/internal/database/storage/engine"
	"github.com/qwaseri832/DataBase/internal/database/storage/wal"
	"github.com/qwaseri832/DataBase/internal/tools"
)

var (
	ErrNotFound = errors.New("not found")
	ErrReadOnly = errors.New("read-only: mutable operation on slave")
)

type Replica interface {
	IsMaster() bool
}

type Option func(*Storage)

func WithWAL(w *wal.WAL) Option    { return func(s *Storage) { s.wal = w } }
func WithReplica(r Replica) Option { return func(s *Storage) { s.replica = r } }
func WithStream(ch <-chan []wal.Record) Option {
	return func(s *Storage) { s.stream = ch }
}

type Storage struct {
	engine  *engine.Engine
	wal     *wal.WAL
	replica Replica
	stream  <-chan []wal.Record
	idgen   *IDGen
	logger  *zap.Logger
}

func New(eng *engine.Engine, logger *zap.Logger, opts ...Option) (*Storage, error) {
	s := &Storage{engine: eng, logger: logger}
	for _, o := range opts {
		o(s)
	}

	var lastLSN int64
	if s.wal != nil {
		recs, err := s.wal.Recover()
		if err != nil {
			return nil, fmt.Errorf("recover from WAL: %w", err)
		}
		lastLSN = s.applyRecords(recs)
		s.wal.OnFlush(func(recs []wal.Record) { s.applyRecords(recs) })
		logger.Info("recovered from WAL",
			zap.Int("records", len(recs)),
			zap.Int64("last_lsn", lastLSN),
		)
	}

	s.idgen = NewIDGen(lastLSN)
	return s, nil
}

func (s *Storage) ApplyStream(ctx context.Context) {
	if s.stream == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case recs, ok := <-s.stream:
			if !ok {
				return
			}
			s.applyRecords(recs)
		}
	}
}

func (s *Storage) Set(ctx context.Context, key, val string) error {
	return s.mutate(ctx, wal.OpSet, []string{key, val}, func(ctx context.Context) {
		s.engine.Set(ctx, key, val)
	})
}

func (s *Storage) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	v, ok := s.engine.Get(ctx, key)
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *Storage) Del(ctx context.Context, key string) error {
	return s.mutate(ctx, wal.OpDel, []string{key}, func(ctx context.Context) {
		s.engine.Del(ctx, key)
	})
}

func (s *Storage) mutate(ctx context.Context, op wal.Op, args []string, apply func(context.Context)) error {
	if s.replica != nil && !s.replica.IsMaster() {
		return ErrReadOnly
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if s.wal == nil {
		apply(tools.WithTxID(ctx, s.idgen.Next()))
		return nil
	}

	res, err := s.wal.Append(op, args).AwaitContext(ctx)
	if err != nil {
		return err
	}
	return res
}

func (s *Storage) applyRecords(recs []wal.Record) int64 {
	var last int64
	for _, r := range recs {
		if r.LSN > last {
			last = r.LSN
		}

		if want := r.Op.ArgCount(); len(r.Args) < want {
			s.logger.Warn("skipped malformed WAL record",
				zap.Int64("lsn", r.LSN),
				zap.Stringer("op", r.Op),
				zap.Int("args", len(r.Args)),
				zap.Int("want", want),
			)
			continue
		}

		ctx := tools.WithTxID(context.Background(), r.LSN)
		switch r.Op {
		case wal.OpSet:
			s.engine.Set(ctx, r.Args[0], r.Args[1])
		case wal.OpDel:
			s.engine.Del(ctx, r.Args[0])
		default:
			s.logger.Warn("unknown command in WAL",
				zap.Int64("lsn", r.LSN),
				zap.Stringer("op", r.Op),
			)
		}
	}
	return last
}

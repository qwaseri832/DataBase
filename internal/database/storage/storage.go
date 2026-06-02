package storage

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"spider/internal/database/compute"
	"spider/internal/database/storage/engine"
	"spider/internal/database/storage/wal"
	"spider/internal/tools"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrReadOnly  = errors.New("read-only: mutable operation on slave")
)

// Replica определяет роль узла.
type Replica interface {
	IsMaster() bool
}

// Option настраивает Storage.
type Option func(*Storage)

func WithWAL(w *wal.WAL) Option      { return func(s *Storage) { s.wal = w } }
func WithReplica(r Replica) Option   { return func(s *Storage) { s.replica = r } }
func WithStream(ch <-chan []wal.Record) Option {
	return func(s *Storage) { s.stream = ch }
}

// Storage связывает Engine, WAL и Replication.
type Storage struct {
	engine  *engine.Engine
	wal     *wal.WAL
	replica Replica
	stream  <-chan []wal.Record
	idgen   *IDGen
	logger  *zap.Logger
}

func New(eng *engine.Engine, logger *zap.Logger, opts ...Option) *Storage {
	s := &Storage{engine: eng, logger: logger}
	for _, o := range opts {
		o(s)
	}

	var lastLSN int64
	if s.wal != nil {
		recs, err := s.wal.Recover()
		if err != nil {
			logger.Error("wal recover", zap.Error(err))
		} else {
			lastLSN = s.applyRecords(recs)
		}
	}

	if s.stream != nil {
		go func() {
			for recs := range s.stream {
				s.applyRecords(recs)
			}
		}()
	}

	s.idgen = NewIDGen(lastLSN)
	return s
}

func (s *Storage) Set(ctx context.Context, key, val string) error {
	if s.replica != nil && !s.replica.IsMaster() {
		return ErrReadOnly
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	txID := s.idgen.Next()
	ctx = tools.WithTxID(ctx, txID)

	if s.wal != nil {
		if err := s.wal.Append(txID, compute.CmdSet, []string{key, val}).Await(); err != nil {
			return err
		}
	}

	s.engine.Set(ctx, key, val)
	return nil
}

func (s *Storage) Get(ctx context.Context, key string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	txID := s.idgen.Next()
	ctx = tools.WithTxID(ctx, txID)

	v, ok := s.engine.Get(ctx, key)
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (s *Storage) Del(ctx context.Context, key string) error {
	if s.replica != nil && !s.replica.IsMaster() {
		return ErrReadOnly
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	txID := s.idgen.Next()
	ctx = tools.WithTxID(ctx, txID)

	if s.wal != nil {
		if err := s.wal.Append(txID, compute.CmdDel, []string{key}).Await(); err != nil {
			return err
		}
	}

	s.engine.Del(ctx, key)
	return nil
}

func (s *Storage) applyRecords(recs []wal.Record) int64 {
	var last int64
	for _, r := range recs {
		if r.LSN > last {
			last = r.LSN
		}
		ctx := tools.WithTxID(context.Background(), r.LSN)
		switch r.Cmd {
		case compute.CmdSet:
			s.engine.Set(ctx, r.Args[0], r.Args[1])
		case compute.CmdDel:
			s.engine.Del(ctx, r.Args[0])
		}
	}
	return last
}

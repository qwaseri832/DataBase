package engine

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/qwaseri832/DataBase/internal/tools"
)

type Option func(*Engine)

func WithPartitions(n int) Option {
	return func(e *Engine) {
		if n < 1 {
			n = 1
		}
		e.parts = make([]*HashTable, n)
		for i := range e.parts {
			e.parts[i] = newHashTable()
		}
	}
}

type Engine struct {
	parts  []*HashTable
	logger *zap.Logger
}

func New(logger *zap.Logger, opts ...Option) *Engine {
	e := &Engine{logger: logger}
	for _, o := range opts {
		o(e)
	}
	if len(e.parts) == 0 {
		e.parts = []*HashTable{newHashTable()}
	}
	return e
}

func (e *Engine) Set(ctx context.Context, key, value string) {
	e.partition(key).Set(key, value)
	e.debug(ctx, "set", key, false, false)
}

func (e *Engine) Get(ctx context.Context, key string) (string, bool) {
	v, ok := e.partition(key).Get(key)
	e.debug(ctx, "get", key, true, ok)
	return v, ok
}

func (e *Engine) Del(ctx context.Context, key string) {
	e.partition(key).Del(key)
	e.debug(ctx, "del", key, false, false)
}

func (e *Engine) debug(ctx context.Context, msg, key string, withFound, found bool) {
	ce := e.logger.Check(zapcore.DebugLevel, msg)
	if ce == nil {
		return
	}
	fields := []zap.Field{
		zap.Int64("tx", tools.TxID(ctx)),
		zap.String("key", key),
	}
	if withFound {
		fields = append(fields, zap.Bool("found", found))
	}
	ce.Write(fields...)
}

func (e *Engine) partition(key string) *HashTable {
	if len(e.parts) == 1 {
		return e.parts[0]
	}
	return e.parts[fnv1a(key)%uint32(len(e.parts))]
}

func fnv1a(s string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)
	h := uint32(offset32)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return h
}

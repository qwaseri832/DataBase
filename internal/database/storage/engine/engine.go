package engine

import (
	"context"
	"hash/fnv"

	"go.uber.org/zap"

	"spider/internal/tools"
)

// Option позволяет настроить Engine при создании.
type Option func(*Engine)

// WithPartitions задаёт количество партиций.
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

// Engine — in-memory хранилище с партиционированием по FNV-1a.
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
	p := e.partition(key)
	p.Set(key, value)
	e.logger.Debug("set", zap.Int64("tx", tools.TxID(ctx)), zap.String("key", key))
}

func (e *Engine) Get(ctx context.Context, key string) (string, bool) {
	p := e.partition(key)
	v, ok := p.Get(key)
	e.logger.Debug("get", zap.Int64("tx", tools.TxID(ctx)), zap.String("key", key), zap.Bool("found", ok))
	return v, ok
}

func (e *Engine) Del(ctx context.Context, key string) {
	p := e.partition(key)
	p.Del(key)
	e.logger.Debug("del", zap.Int64("tx", tools.TxID(ctx)), zap.String("key", key))
}

func (e *Engine) partition(key string) *HashTable {
	if len(e.parts) == 1 {
		return e.parts[0]
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return e.parts[int(h.Sum32())%len(e.parts)]
}

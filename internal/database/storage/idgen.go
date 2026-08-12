package storage

import "sync/atomic"

type IDGen struct {
	counter atomic.Int64
}

func NewIDGen(start int64) *IDGen {
	g := &IDGen{}
	g.counter.Store(start)
	return g
}

func (g *IDGen) Next() int64 {
	return g.counter.Add(1)
}

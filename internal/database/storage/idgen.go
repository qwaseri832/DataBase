package storage

import (
	"math"
	"sync/atomic"
)

// IDGen генерирует монотонно растущие ID (LSN).
type IDGen struct {
	counter atomic.Int64
}

func NewIDGen(start int64) *IDGen {
	g := &IDGen{}
	g.counter.Store(start)
	return g
}

func (g *IDGen) Next() int64 {
	g.counter.CompareAndSwap(math.MaxInt64, 0)
	return g.counter.Add(1)
}

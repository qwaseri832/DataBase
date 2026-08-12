package syncx

import (
	"context"
	"sync"
)

type Promise[T any] struct {
	done chan struct{}
	once sync.Once
	val  T
}

func NewPromise[T any]() *Promise[T] {
	return &Promise[T]{done: make(chan struct{})}
}

func (p *Promise[T]) Resolve(val T) {
	p.once.Do(func() {
		p.val = val
		close(p.done)
	})
}

func (p *Promise[T]) Future() Future[T] {
	return Future[T]{p: p}
}

type Future[T any] struct {
	p *Promise[T]
}

func (f Future[T]) Await() T {
	if f.p == nil {
		var zero T
		return zero
	}
	<-f.p.done
	return f.p.val
}

func (f Future[T]) AwaitContext(ctx context.Context) (T, error) {
	var zero T
	if f.p == nil {
		return zero, nil
	}
	select {
	case <-f.p.done:
		return f.p.val, nil
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

package syncx

// Future позволяет получить результат, который будет установлен позже.
type Future[T any] struct {
	ch <-chan T
}

// Await блокируется до получения значения.
func (f Future[T]) Await() T {
	return <-f.ch
}

// Promise — парная к Future: устанавливает результат ровно один раз.
type Promise[T any] struct {
	ch   chan T
	done bool
}

func NewPromise[T any]() *Promise[T] {
	return &Promise[T]{ch: make(chan T, 1)}
}

// Resolve записывает значение. Повторный вызов игнорируется.
func (p *Promise[T]) Resolve(val T) {
	if p.done {
		return
	}
	p.done = true
	p.ch <- val
	close(p.ch)
}

// Future возвращает Future, привязанный к этому Promise.
func (p *Promise[T]) Future() Future[T] {
	return Future[T]{ch: p.ch}
}

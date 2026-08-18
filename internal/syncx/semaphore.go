package syncx

import "context"

type Semaphore struct {
	slots chan struct{}
}

func NewSemaphore(n int) *Semaphore {
	if n <= 0 {
		return &Semaphore{}
	}
	return &Semaphore{slots: make(chan struct{}, n)}
}

func (s *Semaphore) AcquireContext(ctx context.Context) bool {
	if s.slots == nil {
		return true
	}
	select {
	case s.slots <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Semaphore) Release() {
	if s.slots == nil {
		return
	}
	<-s.slots
}

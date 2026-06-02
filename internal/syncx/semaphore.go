package syncx

// Semaphore ограничивает число одновременных операций.
type Semaphore struct {
	slots chan struct{}
}

func NewSemaphore(n int) *Semaphore {
	if n <= 0 {
		return &Semaphore{}
	}
	return &Semaphore{slots: make(chan struct{}, n)}
}

func (s *Semaphore) Acquire() {
	if s.slots == nil {
		return
	}
	s.slots <- struct{}{}
}

func (s *Semaphore) Release() {
	if s.slots == nil {
		return
	}
	<-s.slots
}

package util

type RingBuffer[T any] struct {
	ch chan T
}

func CreateRingBuffer[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		ch: make(chan T, size),
	}
}

func (r *RingBuffer[T]) Push(data T) {
	select {
	case r.ch <- data:
	default:
		<-r.ch
		r.ch <- data
	}
}

func (r *RingBuffer[T]) Read() <-chan T {
	return r.ch
}

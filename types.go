package main

type ringBuffer[T any] struct {
	ch chan T
}

func createRingBuffer[T any](size int) *ringBuffer[T] {
	return &ringBuffer[T]{
		ch: make(chan T, size),
	}
}

func (r *ringBuffer[T]) Push(data T) {
	select {
	case r.ch <- data:
	default:
		<-r.ch
		r.ch <- data
	}
}

func (r *ringBuffer[T]) Read() <-chan T {
	return r.ch
}

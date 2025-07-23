package main

type client struct {
	id             int
	audio_chain_ch *ringBuffer
}

type ringBuffer struct {
	ch chan []float32
}

func createRingBuffer(size int) *ringBuffer {
	return &ringBuffer{
		ch: make(chan []float32, size),
	}
}

func (r *ringBuffer) Push(data []float32) {
	select {
	case r.ch <- data:
	default:
		<-r.ch
		r.ch <- data
	}
}

func (r *ringBuffer) Read() <-chan []float32 {
	return r.ch
}

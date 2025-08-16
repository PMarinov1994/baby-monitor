package main

import (
	"context"

	"github.com/pmarinov1994/go4vl/v4l2"
)

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

// Streaming Device
type StreamingDevice struct {
	fd      uintptr
	name    string
	bufType v4l2.BufType
	buffers [][]byte
	count   uint32
	cap     v4l2.Capability
	ioType  v4l2.IOType

	output <-chan []byte
	input  <-chan []byte
}

func (dev StreamingDevice) BufferType() v4l2.BufType {
	return dev.bufType
}

func (dev StreamingDevice) Fd() uintptr {
	return dev.fd
}

func (dev StreamingDevice) MemIOType() v4l2.IOType {
	return dev.ioType
}

func (dev StreamingDevice) Buffers() [][]byte {
	return dev.buffers
}

func (dev StreamingDevice) BufferCount() uint32 {
	return dev.count
}

func (dev StreamingDevice) Capability() v4l2.Capability {
	return dev.cap
}

func (dev StreamingDevice) Name() string {
	return dev.name
}

func (dev StreamingDevice) GetOutput() <-chan []byte {
	return dev.output
}

func (dev StreamingDevice) SetInput(input <-chan []byte) {
}

func (dev StreamingDevice) Start(context.Context) error {
	return nil
}

func (dev StreamingDevice) Stop() error {
	return nil
}

func (dev StreamingDevice) Close() error {
	return nil
}
